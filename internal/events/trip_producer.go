package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type EventType string

const (
	EventTypeTripCreated            EventType = "trip_created"
	EventTypeTripPreapproved        EventType = "trip_preapproved"
	EventTypeTripPublished          EventType = "trip_published"
	EventTypeTripStarted            EventType = "trip_started"
	EventTypeTripCanceled           EventType = "trip_canceled"
	EventTypeTripCompleted          EventType = "trip_completed"
	EventTypeTripPublishNotificated EventType = "trip_publish_notificated"
)

type TripCreated struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type TripPreapproved struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type TripPublished struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type TripStarted struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

func NewTripCreated(
	eventID uuid.UUID,
	tripID, driverID, companyID uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
) TripCreated {
	return TripCreated{
		EventID:       eventID.String(),
		EventType:     EventTypeTripCreated,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		CompanyID:     companyID.String(),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func (e TripCreated) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

func NewTripPreapproved(
	eventID uuid.UUID,
	tripID, driverID, companyID uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
) TripPreapproved {
	return TripPreapproved{
		EventID:       eventID.String(),
		EventType:     EventTypeTripPreapproved,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		CompanyID:     companyID.String(),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func (e TripPreapproved) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

func NewTripPublished(
	eventID uuid.UUID,
	tripID, driverID, companyID uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
) TripPublished {
	return TripPublished{
		EventID:       eventID.String(),
		EventType:     EventTypeTripPublished,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		CompanyID:     companyID.String(),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func (e TripPublished) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

func NewTripStarted(
	eventID uuid.UUID,
	tripID, driverID, companyID uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
) TripStarted {
	return TripStarted{
		EventID:       eventID.String(),
		EventType:     EventTypeTripStarted,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		CompanyID:     companyID.String(),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func (e TripStarted) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

type TripCanceled struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	PassengerIDs  []string  `json:"passenger_ids"`
	RequestIDs    []string  `json:"request_ids"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type TripPublishNotificated struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	FromPoint     string    `json:"from_point"`
	ToPoint       string    `json:"to_point"`
	DepartureTime time.Time `json:"departure_time"`
	PassengerIDs  []string  `json:"passenger_ids"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type TripCompleted struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	TripID        string    `json:"trip_id"`
	DriverID      string    `json:"driver_id"`
	CompanyID     string    `json:"company_id"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Traceparent   string    `json:"traceparent,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}

func NewTripCanceled(
	eventID uuid.UUID,
	tripID, driverID, companyID uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
	passengerIDs []string,
	requestIDs []string,
) TripCanceled {
	return TripCanceled{
		EventID:       eventID.String(),
		EventType:     EventTypeTripCanceled,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		CompanyID:     companyID.String(),
		PassengerIDs:  nonNilStrings(passengerIDs),
		RequestIDs:    nonNilStrings(requestIDs),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func NewTripPublishNotificated(
	eventID uuid.UUID,
	tripID, driverID uuid.UUID,
	fromPoint, toPoint string,
	departureTime time.Time,
	passengerIDs []string,
	occurredAt time.Time,
	meta EventMetadata,
) TripPublishNotificated {
	return TripPublishNotificated{
		EventID:       eventID.String(),
		EventType:     EventTypeTripPublishNotificated,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		FromPoint:     fromPoint,
		ToPoint:       toPoint,
		DepartureTime: departureTime,
		PassengerIDs:  nonNilStrings(passengerIDs),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (e TripCanceled) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

func (e TripPublishNotificated) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

func NewTripCompleted(
	eventID uuid.UUID,
	tripID, driverID, companyID uuid.UUID,
	occurredAt time.Time,
	meta EventMetadata,
) TripCompleted {
	return TripCompleted{
		EventID:       eventID.String(),
		EventType:     EventTypeTripCompleted,
		TripID:        tripID.String(),
		DriverID:      driverID.String(),
		CompanyID:     companyID.String(),
		CorrelationID: meta.CorrelationID,
		CausationID:   meta.CausationID,
		Traceparent:   meta.Traceparent,
		OccurredAt:    occurredAt,
	}
}

func (e TripCompleted) Metadata() EventMetadata {
	return EventMetadata{
		EventID:       e.EventID,
		EventType:     e.EventType,
		CorrelationID: e.CorrelationID,
		CausationID:   e.CausationID,
		Traceparent:   e.Traceparent,
		OccurredAt:    e.OccurredAt,
	}
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
		},
	}
}

func (p *Producer) PublishTripCreated(ctx context.Context, event TripCreated) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) PublishTripPreapproved(ctx context.Context, event TripPreapproved) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) PublishTripPublished(ctx context.Context, event TripPublished) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) PublishTripStarted(ctx context.Context, event TripStarted) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) PublishTripCanceled(ctx context.Context, event TripCanceled) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) PublishTripCompleted(ctx context.Context, event TripCompleted) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) PublishTripPublishNotificated(ctx context.Context, event TripPublishNotificated) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.TripID),
		Value:   data,
		Headers: event.Metadata().KafkaHeaders(),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
