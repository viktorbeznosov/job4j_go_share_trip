package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"go.opentelemetry.io/otel"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/entity"
)

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
			return nil, tripErrors.ErrTripNotFound
		}
		return nil, fmt.Errorf("failed to get trip by id: %w", err)
	}

	return &trip, nil
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
			return entity.Trip{}, tripErrors.ErrTripNotFound
		}
		return entity.Trip{}, fmt.Errorf("failed to get trip by id for update: %w", err)
	}

	return trip, nil
}
