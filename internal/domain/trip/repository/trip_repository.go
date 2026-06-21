package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j_go_share_trip/internal/domain/trip/entity"
)

var ErrTripNotFound = errors.New("trip not found")

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, trip *entity.Trip) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
    defer func() {
        if err != nil {
            if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
                log.Printf("failed to rollback transaction: %v", rollbackErr)
            }
        }
    }()

	if err := r.createTrip(ctx, tx, trip); err != nil {
		return err
	}

	if err := r.createHistory(ctx, tx, trip.ID, nil, &trip.Status); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) createTrip(ctx context.Context, db Querier, trip *entity.Trip) error {
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

	return err
}

func (r *Repository) CreateHistory(ctx context.Context, tripID uuid.UUID, fromStatus *entity.Status, toStatus *entity.Status) error {
	return r.createHistory(ctx, r.db, tripID, fromStatus, toStatus)
}

func (r *Repository) createHistory(ctx context.Context, db Querier, tripID uuid.UUID, fromStatus *entity.Status, toStatus *entity.Status) error {
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
	return err
}

func (r *Repository) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
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


