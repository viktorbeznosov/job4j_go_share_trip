package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

const (
	EventTypeRequestCreated      EventType = "request_created"
	EventTypeRequestPublished    EventType = "request_published"
	EventTypeRequestPreconfirmed EventType = "request_preconfirmed"
	EventTypeRequestConfirmed    EventType = "request_confirmed"
	EventTypeRequestRejected     EventType = "request_rejected"
	EventTypeRequestStarted      EventType = "request_started"
	EventTypeRequestCompleted    EventType = "request_completed"
	EventTypeRequestExpired      EventType = "request_expired"
)

type RequestEvent struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	RequestID     string    `json:"request_id"`
	PassengerID   string    `json:"passenger_id"`
	DriverID      string    `json:"driver_id,omitempty"`
	TripID        string    `json:"trip_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

func NewRequestEvent(
	eventType EventType,
	eventID uuid.UUID,
	requestID uuid.UUID,
	passengerID uuid.UUID,
	tripID *uuid.UUID,
	driverID *uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
) RequestEvent {
	event := RequestEvent{
		EventID:       eventID.String(),
		EventType:     eventType,
		RequestID:     requestID.String(),
		PassengerID:   passengerID.String(),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
	if tripID != nil {
		event.TripID = tripID.String()
	}
	if driverID != nil {
		event.DriverID = driverID.String()
	}

	return event
}

func (e RequestEvent) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

func (e *RequestEvent) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}

func (p *Producer) PublishRequestEvent(ctx context.Context, event RequestEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.RequestID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}
