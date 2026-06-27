// internal/platform/outbox/event.go
package outbox

import (
	"time"

	"github.com/google/uuid"
)

type EventName string

const (
    TripCreated EventName = "trip_created"
    TripPublished EventName = "trip_published"
)

type Event struct {
	ID          uuid.UUID
	EventName   EventName
	AggregateID uuid.UUID
	Payload     []byte
	CreatedAt   time.Time
}