package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type MoveFromDraftToPublishRequest struct {
	Trip      GetTripResponse
	ClientID  uuid.UUID
	CompanyID uuid.UUID
	OldStatus string
	NewStatus string
}

type MoveFromDraftToPublishResponse struct {
	ID            uuid.UUID
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        string
	CreatedAt     time.Time
}

func (s *TripService) MoveFromDraftToPublish(ctx context.Context, req MoveFromDraftToPublishRequest) (*MoveFromDraftToPublishResponse, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "ModeTripFromDraftToPublished"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.Trip.ID.String()),
	)

	ctx = logctx.WithTripID(ctx, req.Trip.ID.String())
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

	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.Update")
	defer span.End()

	started := time.Now()
	result := "success"

	defer func() {
		s.metrics.TripPublishTotal.WithLabelValues(result).Inc()
		s.metrics.TripPublishDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	logger.Info("update trip from draft to publish started")

	domainTrip := domain.GetTripResponse{
		ID:            req.Trip.ID,
		DriverID:      req.Trip.DriverID,
		FromPoint:     req.Trip.FromPoint,
		ToPoint:       req.Trip.ToPoint,
		DepartureTime: req.Trip.DepartureTime,
		Seats:         req.Trip.Seats,
		Status:        req.Trip.Status,
		CreatedAt:     req.Trip.CreatedAt,
	}

	domainReq := domain.MoveFromDraftToPublishRequest{
		Trip:      domainTrip,
		ClientID:  req.ClientID,
		CompanyID: req.CompanyID,
		OldStatus: req.OldStatus,
		NewStatus: req.NewStatus,
	}

	domainResp, err := s.tripDomain.MoveFromDraftToPublish(ctx, domainReq)
	if err != nil {
		logger.Error("failed to move from draft to publish", slog.Any("error", err))
		result = "error"
		return nil, err
	}

	logger.Info("update trip completed", slog.String("new_status", string(domainResp.Status)))

	return &MoveFromDraftToPublishResponse{
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
