package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/storage"
)

var ErrTripNotFound = errors.New("trip not found")

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type RowQuerier interface {
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TripRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *TripRepository {
	return &TripRepository{db: db}
}

func (r *TripRepository) GetDB() *pgxpool.Pool {
    return r.db
}

func (r *TripRepository) Create(ctx context.Context, trip *entity.Trip) error {
    return r.CreateTx(ctx, r.db, trip)
}

func (r *TripRepository) Update(ctx context.Context, trip *entity.Trip, oldStatus entity.Status) (*entity.Trip, error) {
    err := r.UpdateTx(ctx, r.db, trip)
    if (err != nil) {
        return nil, err
    }

    updatedTrip, err := r.GetByTripID(ctx, trip.ID)
    if (err != nil) {
        return nil, err
    }

    return updatedTrip, nil
}

func (r *TripRepository) CreateTx(ctx context.Context, db Querier, trip *entity.Trip) error {
	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "TripRepository"),
		slog.String("operation", "Create"),
		slog.String("trip_id", trip.ID.String()),
		slog.String("client_id", trip.DriverID.String()),
	)

    logger.Info("insert trip started")

	const query = `
		INSERT INTO public.trips (
			id,
			driver_id,
			from_point,
			to_point,
			departure_time,
			seats,
			status,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := db.Exec(
		ctx,
		query,
		trip.ID,
		trip.DriverID,
		trip.FromPoint,
		trip.ToPoint,
		trip.DepartureTime,
		trip.Seats,
		trip.Status,
		trip.CreatedAt,
	)

	if err != nil {
		logger.Error(
			"insert trip failed",
			slog.Any("error", err),
		)
		return fmt.Errorf("tx.Exec create trip: %w", err)
	}

	logger.Info("insert trip completed")

	return nil
}

func (r *TripRepository) UpdateTx(ctx context.Context, db Querier, trip *entity.Trip) error {
	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "TripRepository"),
		slog.String("operation", "Update"),
		slog.String("trip_id", trip.ID.String()),
		slog.String("client_id", trip.DriverID.String()),
	)

    logger.Info("update trip started")

    const query = `
        UPDATE public.trips
        SET status = $1
        WHERE id = $2
    `

    _, err := db.Exec(
        ctx,
        query,
        trip.Status,
        trip.ID,
    )

	if err != nil {
		logger.Error(
			"update trip failed",
			slog.Any("error", err),
		)
		return fmt.Errorf("tx.Exec update trip: %w", err)
	}

	logger.Info("update trip completed")

    return nil
}

func (r *TripRepository) CreateHistoryTx(
	ctx context.Context,
	db Querier,
	tripID uuid.UUID,
	fromStatus *entity.Status,
	toStatus *entity.Status,
) error {
	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "TripRepository"),
		slog.String("operation", "Update"),
		slog.String("trip_id", string(tripID.String())),
		slog.String("from_status", string(*fromStatus)),
		slog.String("to_status", string(*toStatus)),
	)

    logger.Info("history create started")

	if tripID == uuid.Nil {
		return errors.New("trip_id is required")
	}
	if toStatus == nil {
		return errors.New("to_status is required")
	}

	var (
		builder strings.Builder
		fields  []string
		args    []any
	)

	builder.WriteString("INSERT INTO public.trip_history (")

	fields = append(fields, "id")
	args = append(args, uuid.New())

	fields = append(fields, "trip_id")
	args = append(args, tripID)

	fields = append(fields, "to_status")
	args = append(args, *toStatus)

	if fromStatus != nil && *fromStatus != "" {
		fields = append(fields, "from_status")
		args = append(args, *fromStatus)
	}

	placeholders := make([]string, len(args))
	for i := range args {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	builder.WriteString(strings.Join(fields, ", "))
	builder.WriteString(") VALUES (")
	builder.WriteString(strings.Join(placeholders, ", "))
	builder.WriteString(")")

	log.Println(builder.String())

	_, err := db.Exec(ctx, builder.String(), args...)

    if err != nil {
        logger.Error(
            "save trip history failed",
            slog.Any("error", err),
        )
        return fmt.Errorf("tx.Exec update trip: %w", err)
    }

    logger.Info("save trip history completed")

	return nil
}

func (r *TripRepository) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	const query = `
		SELECT
			id,
			driver_id,
			from_point,
			to_point,
			departure_time,
			seats,
			status,
			created_at
		FROM public.trips
		WHERE id = $1
	`

	var trip entity.Trip

	err := r.db.QueryRow(ctx, query, tripID).Scan(
		&trip.ID,
		&trip.DriverID,
		&trip.FromPoint,
		&trip.ToPoint,
		&trip.DepartureTime,
		&trip.Seats,
		&trip.Status,
		&trip.CreatedAt,
	)
	if err != nil {
		// при желании тут можно завернуть pgx.ErrNoRows в доменную ошибку
		return nil, ErrTripNotFound
	}

	return &trip, nil
}

// GetForUpdateByID - получает поездку с блокировкой FOR UPDATE в новой транзакции
func (r *TripRepository) GetForUpdateByID(
	ctx context.Context,
	id uuid.UUID,
) (entity.Trip, error) {
	var trip entity.Trip

	// Используем storage.Tx для создания транзакции
	_, err := storage.Tx(ctx, r.db, func(tx pgx.Tx) (*entity.Trip, error) {
		// Получаем поездку с блокировкой
		updated, err := r.GetForUpdateByIDWithTX(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		trip = updated
		return &trip, nil
	})

	if err != nil {
		return entity.Trip{}, err
	}

	return trip, nil
}

func (r *TripRepository) GetForUpdateByIDWithTX(
	ctx context.Context,
	tx RowQuerier,
	id uuid.UUID,
) (entity.Trip, error) {
	var trip entity.Trip

	query := `
		SELECT
			id,
			driver_id,
			from_point,
			to_point,
			departure_time,
			seats,
			status,
			created_at
		FROM trips
		WHERE id = $1 FOR UPDATE
	`
	err := tx.QueryRow(ctx, query, id).Scan(
		&trip.ID,
		&trip.DriverID,
		&trip.FromPoint,
		&trip.ToPoint,
		&trip.DepartureTime,
		&trip.Seats,
		&trip.Status,
		&trip.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Trip{}, ErrTripNotFound
		}
		return entity.Trip{}, fmt.Errorf(
			"failed to get trip by id for update: %w", err,
		)
	}

	return trip, nil
}



