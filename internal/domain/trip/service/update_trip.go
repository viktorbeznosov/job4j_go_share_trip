package service

import (
	"context"
	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
	"log"
	"log/slog"
	"time"

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

    uow := storage.NewUnitOfWork(s.tripRepository.GetDB())

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "UpdateTrip"),
		slog.String("client_id", trip.DriverID.String()),
	)

    logger.Info("create trip started")

    txLogger := logger.With(
        slog.String("layer", "transaction"),
    )

    if err := uow.Begin(ctx); err != nil {
        return nil, err
    }
    defer func() {
        if err := uow.Rollback(); err != nil {
            log.Printf("failed to rollback transaction: %v", err)
        }
    }()

    if err := s.tripRepository.UpdateTx(ctx, uow.GetTx(), trip); err != nil {
        logger.Error(
            "failed to update trip",
            slog.Any("error", err),
        )
        return nil, err
    }

    if err := s.tripRepository.CreateHistoryTx(ctx, uow.GetTx(), trip.ID, &oldStatus, &trip.Status); err != nil {
        logger.Error(
            "failed to save event trip",
            slog.Any("error", err),
        )
        return nil, err
    }

    event, _ := s.createOutboxEvent(trip, outbox.TripPublished)
    if err := s.eventRepository.SaveTx(ctx, uow.GetTx(), event); err != nil {
        logger.Error(
            "failed to save event trip",
            slog.Any("error", err),
        )
        return nil, err
    }

    updatedTrip, err := s.tripRepository.GetByTripID(ctx, trip.ID)
    if err != nil {
        return nil, err
    }

    if err := uow.Commit(); err != nil {
        return nil, err
    }

    txLogger.Info(
        "transaction completed",
        slog.String("trip_id", trip.ID.String()),
    )

    return updatedTrip, nil
}



