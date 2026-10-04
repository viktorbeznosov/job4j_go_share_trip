package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"

	"job4j_go_share_trip/internal/observability/logctx"
)

const (
	HeaderEventID       = "event_id"
	HeaderEventType     = "event_type"
	HeaderCorrelationID = "correlation_id"
	HeaderCausationID   = "causation_id"
	HeaderTraceparent   = "traceparent"
)

type EventMetadata struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	CorrelationID string    `json:"correlation_id"`
	CausationID   string    `json:"causation_id"`
	Traceparent   string    `json:"traceparent"`
	OccurredAt    time.Time `json:"occurred_at"`
}

func NewEventMetadata(
	eventID string,
	eventType EventType,
	correlationID string,
	causationID string,
	traceparent string,
	occurredAt time.Time,
) EventMetadata {
	return EventMetadata{
		EventID:       eventID,
		EventType:     eventType,
		CorrelationID: correlationID,
		CausationID:   causationID,
		Traceparent:   traceparent,
		OccurredAt:    occurredAt,
	}
}

func MetadataFromContext(
	ctx context.Context,
	eventID string,
	eventType EventType,
	occurredAt time.Time,
) EventMetadata {
	correlationID := logctx.CorrelationID(ctx)
	requestID := logctx.RequestID(ctx)
	if correlationID == "" {
		correlationID = requestID
	}
	causationID := requestID
	if causationID == "" {
		causationID = correlationID
	}

	return NewEventMetadata(
		eventID,
		eventType,
		correlationID,
		causationID,
		logctx.Traceparent(ctx),
		occurredAt,
	)
}

func mergeEventMetadata(base EventMetadata, overlay EventMetadata) EventMetadata {
	if overlay.EventID != "" {
		base.EventID = overlay.EventID
	}
	if overlay.EventType != "" {
		base.EventType = overlay.EventType
	}
	if overlay.CorrelationID != "" {
		base.CorrelationID = overlay.CorrelationID
	}
	if overlay.CausationID != "" {
		base.CausationID = overlay.CausationID
	}
	if overlay.Traceparent != "" {
		base.Traceparent = overlay.Traceparent
	}
	if !overlay.OccurredAt.IsZero() {
		base.OccurredAt = overlay.OccurredAt
	}
	return base
}

func metadataFromJSON(raw []byte) EventMetadata {
	if len(raw) == 0 {
		return EventMetadata{}
	}
	var meta EventMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return EventMetadata{}
	}
	return meta
}

func (m EventMetadata) KafkaHeaders() []kafka.Header {
	return []kafka.Header{
		{Key: HeaderEventID, Value: []byte(m.EventID)},
		{Key: HeaderEventType, Value: []byte(m.EventType)},
		{Key: HeaderCorrelationID, Value: []byte(m.CorrelationID)},
		{Key: HeaderCausationID, Value: []byte(m.CausationID)},
		{Key: HeaderTraceparent, Value: []byte(m.Traceparent)},
	}
}
