package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

type ApproveTripRequest struct {
	TripID    uuid.UUID
	ClientID  uuid.UUID
	CompanyID uuid.UUID
}

type ApproveTripResponse struct {
	ID               uuid.UUID
	DriverID         uuid.UUID
	FromPoint        string
	ToPoint          string
	DepartureTime    time.Time
	Seats            int
	Status           string
	CreatedAt        time.Time
	AlreadyPublished bool
}

func (s *TripService) ApproveTrip(ctx context.Context, req ApproveTripRequest) (*ApproveTripResponse, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.ApproveTrip")
	defer span.End()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "ApproveTrip"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	ctx = logctx.WithTripID(ctx, req.TripID.String())
	ctx = logctx.WithUserID(ctx, req.ClientID.String())

	contractStarted := time.Now()
	contractResult := "success"
	tripPublishAllowedResp, err := s.contractClient.CheckService(
		ctx, req.CompanyID.String(), entity.ServiceTripPublish,
	)
	if err != nil {
		contractResult = "error"
		s.metrics.ContractRequestTotal.WithLabelValues(contractResult).Inc()
		s.metrics.ContractRequestDuration.WithLabelValues(contractResult).
			Observe(time.Since(contractStarted).Seconds())
		logger.Error("failed to check contract service", slog.Any("error", err))
		return nil, fmt.Errorf("%w: %s", tripErrors.ErrTripPublishIsNotAllowed, err.Error())
	}
	if !tripPublishAllowedResp.Allowed {
		contractResult = "denied"
		s.metrics.ContractRequestTotal.WithLabelValues(contractResult).Inc()
		s.metrics.ContractRequestDuration.WithLabelValues(contractResult).
			Observe(time.Since(contractStarted).Seconds())
		return nil, fmt.Errorf("%w: %s", tripErrors.ErrTripPublishIsNotAllowed, tripPublishAllowedResp.Reason)
	}
	s.metrics.ContractRequestTotal.WithLabelValues(contractResult).Inc()
	s.metrics.ContractRequestDuration.WithLabelValues(contractResult).
		Observe(time.Since(contractStarted).Seconds())

	started := time.Now()
	result := "success"

	defer func() {
		s.metrics.TripPublishTotal.WithLabelValues(result).Inc()
		s.metrics.TripPublishDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	logger.Info("approve trip started")

	domainReq := domain.ApproveTripRequest{
		TripID:    req.TripID,
		ClientID:  req.ClientID,
		CompanyID: req.CompanyID,
	}

	domainResp, err := s.tripDomain.ApproveTrip(ctx, domainReq)
	if err != nil {
		logger.Error("failed to approve trip", slog.Any("error", err))
		result = "error"
		return nil, err
	}

	logger.Info("approve trip completed", slog.String("new_status", domainResp.Status))

	return &ApproveTripResponse{
		ID:               domainResp.ID,
		DriverID:         domainResp.DriverID,
		FromPoint:        domainResp.FromPoint,
		ToPoint:          domainResp.ToPoint,
		DepartureTime:    domainResp.DepartureTime,
		Seats:            domainResp.Seats,
		Status:           domainResp.Status,
		CreatedAt:        domainResp.CreatedAt,
		AlreadyPublished: domainResp.AlreadyPublished,
	}, nil
}
