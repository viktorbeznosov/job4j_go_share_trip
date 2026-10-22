package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/observability/logctx"
)

type MoveTripToCompletedRequest struct {
	TripID    uuid.UUID
	ClientID  uuid.UUID
	CompanyID uuid.UUID
}

type MoveTripToCompletedResponse struct {
	ID               uuid.UUID
	DriverID         uuid.UUID
	FromPoint        string
	ToPoint          string
	DepartureTime    time.Time
	Seats            int
	Status           string
	CreatedAt        time.Time
	AlreadyCompleted bool
}

func (s *TripService) MoveToCompleted(
	ctx context.Context,
	req MoveTripToCompletedRequest,
) (*MoveTripToCompletedResponse, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.MoveToCompleted")
	defer span.End()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "MoveToCompleted"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	ctx = logctx.WithTripID(ctx, req.TripID.String())
	ctx = logctx.WithUserID(ctx, req.ClientID.String())

	logger.Info("move trip to completed started")

	domainResp, err := s.tripDomain.MoveToCompleted(ctx, domain.MoveTripToCompletedRequest{
		TripID:    req.TripID,
		ClientID:  req.ClientID,
		CompanyID: req.CompanyID,
	})
	if err != nil {
		logger.Error("failed to move trip to completed", slog.Any("error", err))
		return nil, err
	}

	logger.Info("move trip to completed completed", slog.String("new_status", domainResp.Status))

	return &MoveTripToCompletedResponse{
		ID:               domainResp.ID,
		DriverID:         domainResp.DriverID,
		FromPoint:        domainResp.FromPoint,
		ToPoint:          domainResp.ToPoint,
		DepartureTime:    domainResp.DepartureTime,
		Seats:            domainResp.Seats,
		Status:           domainResp.Status,
		CreatedAt:        domainResp.CreatedAt,
		AlreadyCompleted: domainResp.AlreadyCompleted,
	}, nil
}
