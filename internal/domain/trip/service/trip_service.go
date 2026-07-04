package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/repository"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type TripService struct {
	tripRepository repository.TripRepository
	eventRepository outbox.EventRepository
}

func NewService(
    tripRepository repository.TripRepository,
    eventRepository outbox.EventRepository,
) *TripService {
	return &TripService{
		tripRepository: tripRepository,
		eventRepository: eventRepository,
	}
}

func (s *TripService) Create(ctx context.Context, trip *entity.Trip) error {
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

    return nil
}

func (s *TripService) Update(ctx context.Context, trip *entity.Trip, oldStatus entity.Status) (*entity.Trip, error) {
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

func (s *TripService) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	return s.tripRepository.GetByTripID(ctx, tripID)
}

func (s *TripService) GetForUpdateByID(
    ctx context.Context,
    id uuid.UUID,
) (entity.Trip, error) {
    return s.tripRepository.GetForUpdateByID(ctx, id)
}

func (s *TripService) createOutboxEvent(trip *entity.Trip, eventName outbox.EventName) (*outbox.Event, error) {
    payload, err := json.Marshal(trip)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal trip: %w", err)
    }

    return &outbox.Event{
        ID:          uuid.New(),
        EventName:   eventName,
        AggregateID: trip.ID,
        Payload:     payload,
        CreatedAt:   time.Now(),
    }, nil
}

