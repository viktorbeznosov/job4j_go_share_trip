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

    uow := storage.NewUnitOfWork(s.tripRepository.GetDB())

    txLogger := logger.With(
        slog.String("layer", "transaction"),
    )

    if err := uow.Begin(ctx); err != nil {
        txLogger.Error(
            "failed to begin transaction",
            slog.Any("error", err),
        )
        return err
    }
    defer func() {
        if err := uow.Rollback(); err != nil {
            log.Printf("failed to rollback transaction: %v", err)
        }
    }()

	if err := s.tripRepository.CreateTx(ctx, uow.GetTx(), trip); err != nil {
	    log.Printf("failed to create trip: %v", err)
        logger.Error(
            "failed to create trip",
            slog.Any("error", err),
        )
	    return err
	}

    if err := s.tripRepository.CreateHistoryTx(ctx, uow.GetTx(), trip.ID, nil, &trip.Status); err != nil {
        return err
    }

    event, _ := s.createOutboxEvent(trip, outbox.TripCreated)
    if err := s.eventRepository.SaveTx(ctx, uow.GetTx(), event); err != nil {
        logger.Error(
            "failed to save event trip",
            slog.Any("error", err),
        )
        return err
    }

    if err := uow.Commit(); err != nil {
        return err
    }

    txLogger.Info(
        "transaction completed",
        slog.String("trip_id", trip.ID.String()),
    )

    s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
    s.metrics.TripCreateDuration.WithLabelValues(result).
        Observe(time.Since(started).Seconds())

    return nil
}

