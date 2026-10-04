package outbox

import (
	"time"

	"github.com/google/uuid"
)

type AggregateType string

type EventType string

type Status string

const (
	AggregateTypeTrip AggregateType = "trip"
)

const (
	EventTypeTripCreated   EventType = "trip_created"
	EventTypeTripPublished EventType = "trip_published"
	EventTypeTripStarted   EventType = "trip_started"
)

const (
	StatusPending Status = "pending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
)

type Event struct {
	ID            uuid.UUID
	AggregateType AggregateType
	AggregateID   uuid.UUID
	EventType     EventType
	Payload       []byte
	Metadata      []byte
	Status        Status
	Attempts      int
	LastError     *string
	CreatedAt     time.Time
	SentAt        *time.Time
}

func NewPendingEvent(
	eventType EventType,
	aggregateID uuid.UUID,
	payload []byte,
) Event {
	return Event{
		ID:            uuid.New(),
		AggregateType: AggregateTypeTrip,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		Metadata:      []byte("{}"),
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}
}
