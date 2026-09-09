package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type GetTripRequest struct {
	TripID uuid.UUID
}

type GetTripResponse struct {
	ID            uuid.UUID `json:"id"`
	DriverID      uuid.UUID `json:"driverId"`
	FromPoint     string    `json:"fromPoint"`
	ToPoint       string    `json:"toPoint"`
	DepartureTime time.Time `json:"departureTime"`
	Seats         int       `json:"seats"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (d *TripDomain) GetByTripID(ctx context.Context, request GetTripRequest) (*GetTripResponse, error) {
	trip, err := d.tripRepository.GetByTripID(ctx, request.TripID)
	if err != nil {
		return nil, err
	}

	return &GetTripResponse{
		ID:            trip.ID,
		DriverID:      trip.DriverID,
		FromPoint:     trip.FromPoint,
		ToPoint:       trip.ToPoint,
		DepartureTime: trip.DepartureTime,
		Seats:         trip.Seats,
		Status:        string(trip.Status),
		CreatedAt:     trip.CreatedAt,
	}, nil
}



