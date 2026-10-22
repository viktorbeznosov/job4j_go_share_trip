package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/shared/outbox"
)

func (d *TripDomain) saveTripOutboxTx(
	ctx context.Context,
	tx pgx.Tx,
	aggregateID uuid.UUID,
	outboxType outbox.EventType,
	kafkaType events.EventType,
	buildPayload func(eventID uuid.UUID, occurredAt time.Time, meta events.EventMetadata) (any, error),
) error {
	event := outbox.NewPendingEvent(outboxType, aggregateID, nil)
	meta := events.MetadataFromContext(ctx, event.ID.String(), kafkaType, event.CreatedAt)

	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	event.Metadata = metadataJSON

	payloadObj, err := buildPayload(event.ID, event.CreatedAt, meta)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		return err
	}
	event.Payload = payload

	if err := d.eventRepository.SaveTx(ctx, tx, &event); err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}
	return nil
}
