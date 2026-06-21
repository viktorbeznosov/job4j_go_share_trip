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
	StatusMatched   Status = "canceled"
	StatusConfirmed Status = "completed"
)

type Trip struct {
	ID            uuid.UUID
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        Status
	CreatedAt     time.Time
}

func NewTrip(
	driverID uuid.UUID,
	fromPoint string,
	toPoint string,
	departureTime time.Time,
	seats int,
) (*Trip, error) {
	if driverID == uuid.Nil {
		return nil, errors.New("driver id is required")
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

	return &Trip{
		ID:            uuid.New(),
		DriverID:      driverID,
		FromPoint:     fromPoint,
		ToPoint:       toPoint,
		DepartureTime: departureTime,
		Seats:         seats,
		Status:        StatusDraft,
		CreatedAt:     time.Now(),
	}, nil
}

