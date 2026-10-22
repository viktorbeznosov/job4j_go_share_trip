package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusPreAgreed Status = "pre_agreed"
	StatusAgreed    Status = "agreed"
	StatusStarted   Status = "started"
	StatusCanceled  Status = "canceled"
	StatusCompleted Status = "completed"
)

type Request struct {
	ID            uuid.UUID
	PassengerID   uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        Status
	TripID        *uuid.UUID
	CreatedAt     time.Time
}

func NewRequest(
	passengerID uuid.UUID,
	fromPoint string,
	toPoint string,
	departureTime time.Time,
	seats int,
) (*Request, error) {
	if passengerID == uuid.Nil {
		return nil, errors.New("passenger id is required")
	}
	if fromPoint == "" {
		return nil, errors.New("from point is required")
	}
	if toPoint == "" {
		return nil, errors.New("to point is required")
	}
	if departureTime.IsZero() {
		return nil, errors.New("departure time is required")
	}
	if seats <= 0 {
		return nil, errors.New("seats must be greater than 0")
	}

	return &Request{
		ID:            uuid.New(),
		PassengerID:   passengerID,
		FromPoint:     fromPoint,
		ToPoint:       toPoint,
		DepartureTime: departureTime,
		Seats:         seats,
		Status:        StatusDraft,
		CreatedAt:     time.Now(),
	}, nil
}
