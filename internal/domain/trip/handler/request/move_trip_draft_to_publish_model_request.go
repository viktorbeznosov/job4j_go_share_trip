package request

import (
	"errors"
	"job4j_go_share_trip/internal/validators"

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

	if !validators.IsValidUUID(r.TripID.String()) {
		return ErrInvalidTripID
	}

	// 2. Проверка ClientID
	if r.ClientID == uuid.Nil {
		return ErrClientIDRequired
	}
	if !validators.IsValidUUID(r.ClientID.String()) {
		return ErrInvalidClientID
	}

	return nil
}