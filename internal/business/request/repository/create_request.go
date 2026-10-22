package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (r *RequestRepository) Create(ctx context.Context, request *entity.Request) error {
	return r.CreateTx(ctx, r.db, request)
}

func (r *RequestRepository) CreateTx(ctx context.Context, db Querier, request *entity.Request) error {
	started := time.Now()
	result := "success"

	defer func() {
		r.metrics.RepositoryQueryTotal.WithLabelValues("request_create", result).Inc()
		r.metrics.RepositoryQueryDuration.WithLabelValues("request_create", result).
			Observe(time.Since(started).Seconds())
	}()

	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "RequestRepository"),
		slog.String("operation", "Create"),
		slog.String("request_id", request.ID.String()),
		slog.String("client_id", request.PassengerID.String()),
	)

	logger.Info("insert request started")

	const query = `
		INSERT INTO public.trip_requests (
			id,
			passenger_id,
			from_point,
			to_point,
			departure_time,
			seats,
			status,
			trip_id,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := db.Exec(ctx, query,
		request.ID,
		request.PassengerID,
		request.FromPoint,
		request.ToPoint,
		request.DepartureTime,
		request.Seats,
		request.Status,
		request.TripID,
		request.CreatedAt,
	)
	if err != nil {
		result = "error"
		logger.Error("insert request failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec create request: %w", err)
	}

	logger.Info("insert request completed")
	return nil
}
