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

type MoveTripToCanceledRequest struct {
	TripID    uuid.UUID
	ClientID  uuid.UUID
	CompanyID uuid.UUID
}

type MoveTripToCanceledResponse struct {
	ID              uuid.UUID
	DriverID        uuid.UUID
	FromPoint       string
	ToPoint         string
	DepartureTime   time.Time
	Seats           int
	Status          string
	CreatedAt       time.Time
	AlreadyCanceled bool
}

func (s *TripService) MoveToCanceled(
	ctx context.Context,
	req MoveTripToCanceledRequest,
) (*MoveTripToCanceledResponse, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.MoveToCanceled")
	defer span.End()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "MoveToCanceled"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	ctx = logctx.WithTripID(ctx, req.TripID.String())
	ctx = logctx.WithUserID(ctx, req.ClientID.String())

	logger.Info("move trip to canceled started")

	domainResp, err := s.tripDomain.MoveToCanceled(ctx, domain.MoveTripToCanceledRequest{
		TripID:    req.TripID,
		ClientID:  req.ClientID,
		CompanyID: req.CompanyID,
	})
	if err != nil {
		logger.Error("failed to move trip to canceled", slog.Any("error", err))
		return nil, err
	}

	logger.Info("move trip to canceled completed", slog.String("new_status", domainResp.Status))

	return &MoveTripToCanceledResponse{
		ID:              domainResp.ID,
		DriverID:        domainResp.DriverID,
		FromPoint:       domainResp.FromPoint,
		ToPoint:         domainResp.ToPoint,
		DepartureTime:   domainResp.DepartureTime,
		Seats:           domainResp.Seats,
		Status:          domainResp.Status,
		CreatedAt:       domainResp.CreatedAt,
		AlreadyCanceled: domainResp.AlreadyCanceled,
	}, nil
}
