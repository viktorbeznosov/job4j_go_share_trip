package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (r *TripRepository) Create(ctx context.Context, trip *entity.Trip) error {
	return r.CreateTx(ctx, r.db, trip)
}

func (r *TripRepository) CreateTx(ctx context.Context, db Querier, trip *entity.Trip) error {
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


