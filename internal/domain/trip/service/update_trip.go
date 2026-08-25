package service

import (
	"context"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"

	"go.opentelemetry.io/otel"
)

func (s *TripService) Update(ctx context.Context, trip *entity.Trip, oldStatus entity.Status) (*entity.Trip, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.Update")
	defer span.End()

	started := time.Now()
	result := "success"

	defer func() {
		s.metrics.TripPublishTotal.WithLabelValues(result).Inc()
		s.metrics.TripPublishDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "UpdateTrip"),
		slog.String("client_id", trip.DriverID.String()),
		slog.String("trip_id", trip.ID.String()),
	)

	logger.Info("update trip started")

	event, err := s.createOutboxEvent(trip, outbox.TripPublished)
	if err != nil {
		logger.Error("failed to create outbox event", slog.Any("error", err))
		result = "error"
		return nil, err
	}

	if err := s.tripRepository.UpdateWithHistory(ctx, trip, oldStatus, event); err != nil {
		logger.Error("failed to update trip with history", slog.Any("error", err))
		result = "error"
		return nil, err
	}

	updatedTrip, err := s.tripRepository.GetByTripID(ctx, trip.ID)
	if err != nil {
		logger.Error("failed to get updated trip", slog.Any("error", err))
		return nil, err
	}

	logger.Info("update trip completed", slog.String("new_status", string(trip.Status)))

	return updatedTrip, nil
}