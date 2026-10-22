package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (r *RequestRepository) Update(ctx context.Context, request *entity.Request) error {
	return r.UpdateTx(ctx, r.db, request)
}

func (r *RequestRepository) UpdateTx(ctx context.Context, db Querier, request *entity.Request) error {
	started := time.Now()
	result := "success"

	defer func() {
		r.metrics.RepositoryQueryTotal.WithLabelValues("request_update", result).Inc()
		r.metrics.RepositoryQueryDuration.WithLabelValues("request_update", result).
			Observe(time.Since(started).Seconds())
	}()

	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "RequestRepository"),
		slog.String("operation", "Update"),
		slog.String("request_id", request.ID.String()),
		slog.String("client_id", request.PassengerID.String()),
	)

	logger.Info("update request started")

	const query = `
		UPDATE public.trip_requests
		SET status = $1, trip_id = $2
		WHERE id = $3
	`

	_, err := db.Exec(ctx, query, request.Status, request.TripID, request.ID)
	if err != nil {
		result = "error"
		logger.Error("update request failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec update request: %w", err)
	}

	logger.Info("update request completed")
	return nil
}

func (r *RequestRepository) SumAgreedSeatsByTripIDTx(
	ctx context.Context,
	tx RowQuerier,
	tripID uuid.UUID,
	excludeRequestID uuid.UUID,
) (int, error) {
	const query = `
		SELECT COALESCE(SUM(seats), 0)
		FROM public.trip_requests
		WHERE trip_id = $1
		  AND status = $2
		  AND id <> $3
	`

	var total int
	err := tx.QueryRow(ctx, query, tripID, entity.StatusAgreed, excludeRequestID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum agreed request seats: %w", err)
	}

	return total, nil
}
