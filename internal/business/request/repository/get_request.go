package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/entity"
)

func (r *RequestRepository) GetByID(ctx context.Context, requestID uuid.UUID) (*entity.Request, error) {
	tracer := otel.Tracer("RequestRepository")
	_, span := tracer.Start(ctx, "RequestRepository.GetByID")
	defer span.End()

	return r.scanRequest(r.db.QueryRow(ctx, selectRequestQuery, requestID))
}

func (r *RequestRepository) GetForUpdateByIDWithTX(
	ctx context.Context,
	tx RowQuerier,
	id uuid.UUID,
) (entity.Request, error) {
	const query = selectRequestQuery + " FOR UPDATE"

	request, err := r.scanRequest(tx.QueryRow(ctx, query, id))
	if err != nil {
		return entity.Request{}, err
	}

	return *request, nil
}

const selectRequestQuery = `
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
	FROM public.trip_requests
	WHERE id = $1
`

func (r *RequestRepository) scanRequest(row pgx.Row) (*entity.Request, error) {
	var request entity.Request
	err := row.Scan(
		&request.ID,
		&request.PassengerID,
		&request.FromPoint,
		&request.ToPoint,
		&request.DepartureTime,
		&request.Seats,
		&request.Status,
		&request.TripID,
		&request.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tripErrors.ErrRequestNotFound
		}
		return nil, fmt.Errorf("failed to get request by id: %w", err)
	}

	return &request, nil
}
