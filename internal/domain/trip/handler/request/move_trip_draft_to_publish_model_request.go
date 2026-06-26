package request

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrTripIDRequired = errors.New("tripId is required")
	ErrInvalidTripID  = errors.New("tripId must be a valid UUID")
	ErrClientIDRequired = errors.New("clientId is required")
	ErrInvalidClientID  = errors.New("clientId must be a valid UUID")
)

type MoveTripDraftToPublishModelRequest struct {
	TripID   uuid.UUID `json:"tripId"`
	ClientID uuid.UUID `json:"clientId"`
}

// Validate проверяет все поля запроса
func (r *MoveTripDraftToPublishModelRequest) Validate() error {
	// 1. Проверка TripID
	if r.TripID == uuid.Nil {
		return ErrTripIDRequired
	}
	if r.TripID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidTripID
	}

	// 2. Проверка ClientID
	if r.ClientID == uuid.Nil {
		return ErrClientIDRequired
	}
	if r.ClientID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidClientID
	}

	return nil
}