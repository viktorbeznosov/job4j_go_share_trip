package service

import (
	"context"
	"job4j_go_share_trip/internal/business/trip/domain"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
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

func (s *TripService) GetByTripID(ctx context.Context, request GetTripRequest) (*GetTripResponse, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.GetByTripID")
	defer span.End()

    domainReq := domain.GetTripRequest{
        TripID: request.TripID,
    }

	domainResp, err := s.tripDomain.GetByTripID(ctx, domainReq)
	if err != nil {
		return nil, err
	}

	return &GetTripResponse{
		ID:            domainResp.ID,
		DriverID:      domainResp.DriverID,
		FromPoint:     domainResp.FromPoint,
		ToPoint:       domainResp.ToPoint,
		DepartureTime: domainResp.DepartureTime,
		Seats:         domainResp.Seats,
		Status:        domainResp.Status,
		CreatedAt:     domainResp.CreatedAt,
	}, nil
}
