package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"
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
	db      *pgxpool.Pool
	metrics *metrics.Metrics
}

func NewPostgresRepository(db *pgxpool.Pool, metrics *metrics.Metrics) *TripRepository {
	return &TripRepository{
		db:      db,
		metrics: metrics,
	}
}

func (r *TripRepository) GetDB() *pgxpool.Pool {
	return r.db
}

func (r *TripRepository) Create(ctx context.Context, trip *entity.Trip) error {
	return r.createTx(ctx, r.db, trip)
}

func (r *TripRepository) Update(ctx context.Context, trip *entity.Trip) error {
	return r.updateTx(ctx, r.db, trip)
}

func (r *TripRepository) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	tracer := otel.Tracer("TripRepository")
	_, span := tracer.Start(ctx, "TripRepository.GetByID")
	defer span.End()

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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTripNotFound
		}
		return nil, fmt.Errorf("failed to get trip by id: %w", err)
	}

	return &trip, nil
}

func (r *TripRepository) GetForUpdateByID(ctx context.Context, id uuid.UUID) (entity.Trip, error) {
	var trip entity.Trip

	_, err := storage.Tx(ctx, r.db, func(tx pgx.Tx) (*entity.Trip, error) {
		updated, err := r.getForUpdateByIDWithTX(ctx, tx, id)
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

func (r *TripRepository) createTx(ctx context.Context, db Querier, trip *entity.Trip) error {
	started := time.Now()
	result := "success"

	defer func() {
		r.metrics.RepositoryQueryTotal.WithLabelValues("trip_create", result).Inc()
		r.metrics.RepositoryQueryDuration.WithLabelValues("trip_create", result).
			Observe(time.Since(started).Seconds())
	}()

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

	_, err := db.Exec(ctx, query,
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
		logger.Error("insert trip failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec create trip: %w", err)
	}

	logger.Info("insert trip completed")
	return nil
}

func (r *TripRepository) updateTx(ctx context.Context, db Querier, trip *entity.Trip) error {
	tracer := otel.Tracer("TripRepository")
	_, span := tracer.Start(ctx, "TripRepository.UpdateTx")
	defer span.End()

	started := time.Now()
	result := "success"

	defer func() {
		r.metrics.RepositoryQueryTotal.WithLabelValues("trip_update", result).Inc()
		r.metrics.RepositoryQueryDuration.WithLabelValues("trip_update", result).
			Observe(time.Since(started).Seconds())
	}()

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

	_, err := db.Exec(ctx, query, trip.Status, trip.ID)

	if err != nil {
		logger.Error("update trip failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec update trip: %w", err)
	}

	logger.Info("update trip completed")
	return nil
}

func (r *TripRepository) createHistoryTx(
	ctx context.Context,
	db Querier,
	tripID uuid.UUID,
	fromStatus *entity.Status,
	toStatus *entity.Status,
) error {
	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "TripRepository"),
		slog.String("operation", "CreateHistory"),
		slog.String("trip_id", tripID.String()),
	)

	if tripID == uuid.Nil {
		return errors.New("trip_id is required")
	}
	if toStatus == nil {
		return errors.New("to_status is required")
	}

	fromStatusStr := ""
	if fromStatus != nil {
		fromStatusStr = string(*fromStatus)
	}
	toStatusStr := string(*toStatus)

	logger = logger.With(
		slog.String("from_status", fromStatusStr),
		slog.String("to_status", toStatusStr),
	)

	logger.Info("history create started")

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

	logger.Debug("history query", slog.String("query", builder.String()))

	_, err := db.Exec(ctx, builder.String(), args...)
	if err != nil {
		logger.Error("save trip history failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec create history: %w", err)
	}

	logger.Info("save trip history completed")
	return nil
}

func (r *TripRepository) getForUpdateByIDWithTX(
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
		return entity.Trip{}, fmt.Errorf("failed to get trip by id for update: %w", err)
	}

	return trip, nil
}

func (r *TripRepository) CreateWithHistory(
	ctx context.Context,
	trip *entity.Trip,
	event *outbox.Event,
) error {
	logger := logctx.Logger(ctx).With(
		slog.String("repository", "TripRepository"),
		slog.String("operation", "CreateWithHistory"),
	)

	logger.Info("starting create with history transaction")

	_, err := storage.Tx(ctx, r.db, func(tx pgx.Tx) (*struct{}, error) {
		// 1. Создаём поездку
		if err := r.createTx(ctx, tx, trip); err != nil {
			logger.Error("failed to create trip in transaction", slog.Any("error", err))
			return nil, err
		}

		// 2. Создаём запись в истории
		if err := r.createHistoryTx(ctx, tx, trip.ID, nil, &trip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return nil, err
		}

		// 3. Сохраняем outbox событие
		if err := r.saveEventTx(ctx, tx, event); err != nil {
			logger.Error("failed to save event in transaction", slog.Any("error", err))
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		logger.Error("create with history transaction failed", slog.Any("error", err))
		return err
	}

	logger.Info("create with history transaction completed")
	return nil
}

func (r *TripRepository) UpdateWithHistory(
	ctx context.Context,
	trip *entity.Trip,
	oldStatus entity.Status,
	event *outbox.Event,
) error {
	logger := logctx.Logger(ctx).With(
		slog.String("repository", "TripRepository"),
		slog.String("operation", "UpdateWithHistory"),
		slog.String("trip_id", trip.ID.String()),
		slog.String("old_status", string(oldStatus)),
		slog.String("new_status", string(trip.Status)),
	)

	logger.Info("starting update with history transaction")

	_, err := storage.Tx(ctx, r.db, func(tx pgx.Tx) (*struct{}, error) {
		if err := r.updateTx(ctx, tx, trip); err != nil {
			logger.Error("failed to update trip in transaction", slog.Any("error", err))
			return nil, err
		}

		if err := r.createHistoryTx(ctx, tx, trip.ID, &oldStatus, &trip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return nil, err
		}

		if err := r.saveEventTx(ctx, tx, event); err != nil {
			logger.Error("failed to save event in transaction", slog.Any("error", err))
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		logger.Error("update with history transaction failed", slog.Any("error", err))
		return err
	}

	logger.Info("update with history transaction completed")
	return nil
}

func (r *TripRepository) saveEventTx(ctx context.Context, db Querier, event *outbox.Event) error {
	logger := logctx.Logger(ctx).With(
		slog.String("repository", "TripRepository"),
		slog.String("operation", "SaveEvent"),
		slog.String("event_id", event.ID.String()),
		slog.String("event_type", string(event.EventName)),
	)

	logger.Info("saving outbox event")

	const query = `
		INSERT INTO public.outbox_event (
			id,
			event_name,
			aggregate_id,
			payload,
			created_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := db.Exec(ctx, query,
		event.ID,
        event.EventName,
        event.AggregateID,
        event.Payload,
        event.CreatedAt,
	)

	if err != nil {
		logger.Error("failed to save outbox event", slog.Any("error", err))
		return fmt.Errorf("failed to save outbox event: %w", err)
	}

	logger.Info("outbox event saved successfully")
	return nil
}