package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"job4j_go_share_trip/internal/access"
	"job4j_go_share_trip/internal/business/trip/domain"
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

	return mapGetTrip(domainResp), nil
}

func (s *TripService) GetByTripIDForViewer(ctx context.Context, tripID uuid.UUID, viewer access.Viewer) (*GetTripResponse, error) {
	domainResp, err := s.tripDomain.GetByTripIDForViewer(ctx, tripID, viewer)
	if err != nil {
		return nil, err
	}
	return mapGetTrip(domainResp), nil
}

type ListTripsInput struct {
	Viewer     access.Viewer
	DriverID   *uuid.UUID
	ListParams access.ListParams
}

type ListTripsResult struct {
	Items  []GetTripResponse
	Total  int
	Limit  int
	Offset int
}

func (s *TripService) List(ctx context.Context, req ListTripsInput) (*ListTripsResult, error) {
	domainResp, err := s.tripDomain.List(ctx, domain.ListTripsInput{
		Viewer:     req.Viewer,
		DriverID:   req.DriverID,
		ListParams: req.ListParams,
	})
	if err != nil {
		return nil, err
	}

	items := make([]GetTripResponse, 0, len(domainResp.Items))
	for i := range domainResp.Items {
		items = append(items, *mapGetTrip(&domainResp.Items[i]))
	}

	return &ListTripsResult{
		Items:  items,
		Total:  domainResp.Total,
		Limit:  domainResp.Limit,
		Offset: domainResp.Offset,
	}, nil
}

func mapGetTrip(domainResp *domain.GetTripResponse) *GetTripResponse {
	return &GetTripResponse{
		ID:            domainResp.ID,
		DriverID:      domainResp.DriverID,
		FromPoint:     domainResp.FromPoint,
		ToPoint:       domainResp.ToPoint,
		DepartureTime: domainResp.DepartureTime,
		Seats:         domainResp.Seats,
		Status:        domainResp.Status,
		CreatedAt:     domainResp.CreatedAt,
	}
}
