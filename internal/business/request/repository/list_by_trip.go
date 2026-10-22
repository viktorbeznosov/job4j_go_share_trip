package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/business/request/entity"
)

func (r *RequestRepository) ListAttachedForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	tripID uuid.UUID,
) ([]entity.Request, error) {
	const query = selectRequestQueryWithoutID + `
		FROM public.trip_requests
		WHERE trip_id = $1
		  AND status NOT IN ('canceled', 'completed')
		ORDER BY created_at
		FOR UPDATE
	`

	rows, err := tx.Query(ctx, query, tripID)
	if err != nil {
		return nil, fmt.Errorf("list attached requests: %w", err)
	}
	defer rows.Close()

	return scanRequestRows(rows)
}

func (r *RequestRepository) ListExpiredForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
) ([]entity.Request, error) {
	const query = selectRequestQueryWithoutID + `
		FROM public.trip_requests
		WHERE status IN ('draft', 'published', 'pre_agreed', 'agreed')
		  AND (departure_time AT TIME ZONE 'UTC')::date
		      < (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date
		ORDER BY departure_time
		FOR UPDATE SKIP LOCKED
	`

	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list expired requests: %w", err)
	}
	defer rows.Close()

	return scanRequestRows(rows)
}

func (r *RequestRepository) ListMatchingPublishedPassengersTx(
	ctx context.Context,
	tx pgx.Tx,
	tripID uuid.UUID,
	fromPoint, toPoint string,
	departureDate string,
) ([]uuid.UUID, error) {
	const query = `
		SELECT DISTINCT req.passenger_id
		FROM public.trip_requests req
		WHERE req.status = 'published'
		  AND req.trip_id IS NULL
		  AND req.from_point = $1
		  AND req.to_point = $2
		  AND (req.departure_time AT TIME ZONE 'UTC')::date = $3::date
		  AND NOT EXISTS (
		      SELECT 1
		      FROM public.trip_publish_notified n
		      WHERE n.trip_id = $4
		        AND n.passenger_id = req.passenger_id
		  )
		ORDER BY req.passenger_id
	`

	rows, err := tx.Query(ctx, query, fromPoint, toPoint, departureDate, tripID)
	if err != nil {
		return nil, fmt.Errorf("list matching published passengers: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan matching passenger: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *RequestRepository) MarkPublishNotifiedTx(
	ctx context.Context,
	tx pgx.Tx,
	tripID uuid.UUID,
	passengerIDs []uuid.UUID,
) error {
	for _, passengerID := range passengerIDs {
		_, err := tx.Exec(ctx, `
			INSERT INTO public.trip_publish_notified (trip_id, passenger_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, tripID, passengerID)
		if err != nil {
			return fmt.Errorf("mark trip publish notified: %w", err)
		}
	}
	return nil
}

const selectRequestQueryWithoutID = `
	SELECT
		id,
		passenger_id,
		from_point,
		to_point,
		departure_time,
		seats,
		status,
		trip_id,
		created_at
`

func scanRequestRows(rows pgx.Rows) ([]entity.Request, error) {
	items := make([]entity.Request, 0)
	for rows.Next() {
		request, err := scanRequestRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *request)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
