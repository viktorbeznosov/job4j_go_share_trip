package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/business/trip/entity"
)

func (r *TripRepository) ListPublishedForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
) ([]entity.Trip, error) {
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
		WHERE status = 'published'
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list published trips: %w", err)
	}
	defer rows.Close()

	trips := make([]entity.Trip, 0)
	for rows.Next() {
		var trip entity.Trip
		if err := rows.Scan(
			&trip.ID,
			&trip.DriverID,
			&trip.FromPoint,
			&trip.ToPoint,
			&trip.DepartureTime,
			&trip.Seats,
			&trip.Status,
			&trip.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan published trip: %w", err)
		}
		trips = append(trips, trip)
	}
	return trips, rows.Err()
}
