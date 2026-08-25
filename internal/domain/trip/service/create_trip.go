package service

import (
	"context"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
)

func (s *TripService) Create(ctx context.Context, trip *entity.Trip) error {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "CreateTrip"),
		slog.String("client_id", trip.DriverID.String()),
	)

	logger.Info("create trip started")

	event, err := s.createOutboxEvent(trip, outbox.TripCreated)
	if err != nil {
		logger.Error("failed to create outbox event", slog.Any("error", err))
		result = "error"
		s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
		s.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
		return err
	}

	if err := s.tripRepository.CreateWithHistory(ctx, trip, event); err != nil {
		logger.Error("failed to create trip with history", slog.Any("error", err))
		result = "error"
		s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
		s.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
		return err
	}

	logger.Info("create trip completed", slog.String("trip_id", trip.ID.String()))

	s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
	s.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())

	return nil
}