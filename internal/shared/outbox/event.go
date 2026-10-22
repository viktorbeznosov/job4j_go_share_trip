package outbox

import (
	"time"

	"github.com/google/uuid"
)

type AggregateType string

type EventType string

type Status string

const (
	AggregateTypeTrip    AggregateType = "trip"
	AggregateTypeRequest AggregateType = "request"
)

const (
	EventTypeTripCreated            EventType = "trip_created"
	EventTypeTripPreapproved        EventType = "trip_preapproved"
	EventTypeTripPublished          EventType = "trip_published"
	EventTypeTripStarted            EventType = "trip_started"
	EventTypeTripCanceled           EventType = "trip_canceled"
	EventTypeTripCompleted          EventType = "trip_completed"
	EventTypeTripPublishNotificated EventType = "trip_publish_notificated"
	EventTypeRequestCreated         EventType = "request_created"
	EventTypeRequestPublished       EventType = "request_published"
	EventTypeRequestPreconfirmed    EventType = "request_preconfirmed"
	EventTypeRequestConfirmed       EventType = "request_confirmed"
	EventTypeRequestRejected        EventType = "request_rejected"
	EventTypeRequestStarted         EventType = "request_started"
	EventTypeRequestCompleted       EventType = "request_completed"
	EventTypeRequestExpired         EventType = "request_expired"
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
	return NewPendingAggregateEvent(AggregateTypeTrip, eventType, aggregateID, payload)
}

func NewPendingAggregateEvent(
	aggregateType AggregateType,
	eventType EventType,
	aggregateID uuid.UUID,
	payload []byte,
) Event {
	return Event{
		ID:            uuid.New(),
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		Metadata:      []byte("{}"),
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}
}
