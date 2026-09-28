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
	EventTypeTripPublished EventType = "trip_published"
	EventTypeTripStarted   EventType = "trip_started"
)

type TripPublished struct {
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type"`
	TripID     string    `json:"trip_id"`
	DriverID   string    `json:"driver_id"`
	CompanyID  string    `json:"company_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

type TripStarted struct {
	EventID    string    `json:"event_id"`
	EventType  EventType `json:"event_type"`
	TripID     string    `json:"trip_id"`
	DriverID   string    `json:"driver_id"`
	CompanyID  string    `json:"company_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func NewTripPublished(eventID uuid.UUID, tripID, driverID, companyID uuid.UUID, occurredAt time.Time) TripPublished {
	return TripPublished{
		EventID:    eventID.String(),
		EventType:  EventTypeTripPublished,
		TripID:     tripID.String(),
		DriverID:   driverID.String(),
		CompanyID:  companyID.String(),
		OccurredAt: occurredAt,
	}
}

func NewTripStarted(eventID uuid.UUID, tripID, driverID, companyID uuid.UUID, occurredAt time.Time) TripStarted {
	return TripStarted{
		EventID:    eventID.String(),
		EventType:  EventTypeTripStarted,
		TripID:     tripID.String(),
		DriverID:   driverID.String(),
		CompanyID:  companyID.String(),
		OccurredAt: occurredAt,
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

func (p *Producer) PublishTripPublished(ctx context.Context, event TripPublished) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.TripID),
		Value: data,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
