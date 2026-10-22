package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/shared/outbox"
)

func (d *TripDomain) listAttachedRequestsTx(
	ctx context.Context,
	tx pgx.Tx,
	tripID uuid.UUID,
) ([]entity.Request, error) {
	return d.requestRepository.ListAttachedForUpdateTx(ctx, tx, tripID)
}

func (d *TripDomain) detachRequestsToPublishedTx(
	ctx context.Context,
	tx pgx.Tx,
	requests []entity.Request,
) (passengerIDs []string, requestIDs []string, err error) {
	passengerIDs = make([]string, 0, len(requests))
	requestIDs = make([]string, 0, len(requests))

	for i := range requests {
		req := requests[i]
		oldStatus := req.Status
		req.Status = entity.StatusPublished
		req.TripID = nil

		if err := d.requestRepository.UpdateTx(ctx, tx, &req); err != nil {
			return nil, nil, err
		}
		if err := d.requestRepository.CreateHistoryTx(ctx, tx, req.ID, &oldStatus, &req.Status); err != nil {
			return nil, nil, err
		}

		passengerIDs = append(passengerIDs, req.PassengerID.String())
		requestIDs = append(requestIDs, req.ID.String())
	}

	return passengerIDs, requestIDs, nil
}

func (d *TripDomain) moveAttachedRequestsTx(
	ctx context.Context,
	tx pgx.Tx,
	requests []entity.Request,
	from entity.Status,
	to entity.Status,
	outboxType outbox.EventType,
	kafkaType events.EventType,
) error {
	for i := range requests {
		req := requests[i]
		if req.Status != from {
			continue
		}
		oldStatus := req.Status
		req.Status = to
		if err := d.requestRepository.UpdateTx(ctx, tx, &req); err != nil {
			return err
		}
		if err := d.requestRepository.CreateHistoryTx(ctx, tx, req.ID, &oldStatus, &req.Status); err != nil {
			return err
		}
		if err := d.saveRequestOutboxTx(ctx, tx, outboxType, kafkaType, &req); err != nil {
			return err
		}
	}
	return nil
}

func (d *TripDomain) saveRequestOutboxTx(
	ctx context.Context,
	tx pgx.Tx,
	eventType outbox.EventType,
	kafkaEventType events.EventType,
	request *entity.Request,
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
		nil,
		event.CreatedAt,
		meta,
	))
	if err != nil {
		return err
	}
	event.Payload = payload

	if err := d.eventRepository.SaveTx(ctx, tx, &event); err != nil {
		return fmt.Errorf("failed to save request outbox event: %w", err)
	}
	return nil
}

func filterByStatus(requests []entity.Request, status entity.Status) []entity.Request {
	filtered := make([]entity.Request, 0)
	for i := range requests {
		if requests[i].Status == status {
			filtered = append(filtered, requests[i])
		}
	}
	return filtered
}
