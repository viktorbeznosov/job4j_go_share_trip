package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/entity"
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	tripRepo "job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/shared/outbox"
)

type RequestDomain struct {
	requestRepository requestRepo.RequestRepository
	tripRepository    tripRepo.TripRepository
	eventRepository   outbox.EventRepository
}

func NewDomain(
	requestRepository requestRepo.RequestRepository,
	tripRepository tripRepo.TripRepository,
	eventRepository outbox.EventRepository,
) *RequestDomain {
	return &RequestDomain{
		requestRepository: requestRepository,
		tripRepository:    tripRepository,
		eventRepository:   eventRepository,
	}
}

type RequestResponse struct {
	ID            uuid.UUID
	PassengerID   uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        string
	TripID        *uuid.UUID
	CreatedAt     time.Time
}

func newRequestResponse(request *entity.Request) *RequestResponse {
	return &RequestResponse{
		ID:            request.ID,
		PassengerID:   request.PassengerID,
		FromPoint:     request.FromPoint,
		ToPoint:       request.ToPoint,
		DepartureTime: request.DepartureTime,
		Seats:         request.Seats,
		Status:        string(request.Status),
		TripID:        request.TripID,
		CreatedAt:     request.CreatedAt,
	}
}

func (d *RequestDomain) saveOutboxTx(
	ctx context.Context,
	tx pgx.Tx,
	eventType outbox.EventType,
	kafkaEventType events.EventType,
	request *entity.Request,
	driverID *uuid.UUID,
) error {
	event := outbox.NewPendingAggregateEvent(outbox.AggregateTypeRequest, eventType, request.ID, nil)
	meta := events.MetadataFromContext(ctx, event.ID.String(), kafkaEventType, event.CreatedAt)
	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	event.Metadata = metadataJSON

	payload, err := json.Marshal(events.NewRequestEvent(
		kafkaEventType,
		event.ID,
		request.ID,
		request.PassengerID,
		request.TripID,
		driverID,
		event.CreatedAt,
		meta,
	))
	if err != nil {
		return err
	}
	event.Payload = payload

	if err := d.eventRepository.SaveTx(ctx, tx, &event); err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}

	return nil
}

func (d *RequestDomain) ensureCapacityTx(
	ctx context.Context,
	tx pgx.Tx,
	tripSeats int,
	tripID uuid.UUID,
	request *entity.Request,
) error {
	sumAgreed, err := d.requestRepository.SumAgreedSeatsByTripIDTx(ctx, tx, tripID, request.ID)
	if err != nil {
		return err
	}
	if request.Seats > tripSeats-sumAgreed {
		return fmt.Errorf(
			"%w: request seats %d, trip seats %d, agreed seats %d",
			tripErrors.ErrRequestNotEnoughSeats,
			request.Seats,
			tripSeats,
			sumAgreed,
		)
	}

	return nil
}
