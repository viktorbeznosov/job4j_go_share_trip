package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"

	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (r *TripRepository) Update(ctx context.Context, trip *entity.Trip) error {
	return r.UpdateTx(ctx, r.db, trip)
}

func (r *TripRepository) UpdateTx(ctx context.Context, db Querier, trip *entity.Trip) error {
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
