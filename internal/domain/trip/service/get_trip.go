package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/shared/outbox"
)


func (s *TripService) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.GetByTripID")
	defer span.End()

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

