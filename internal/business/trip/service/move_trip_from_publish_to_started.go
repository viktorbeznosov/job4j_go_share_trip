package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	"job4j_go_share_trip/config"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

type MoveFromPublishToStartedRequest struct {
	TripID    uuid.UUID
	DriverID  uuid.UUID
	OldStatus string
	NewStatus string
}

type MoveFromPublishToStartedResponse struct {
	ID            uuid.UUID
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        string
	CreatedAt     time.Time
}

func (s *TripService) MoveFromPublishToStarted(ctx context.Context, req MoveFromPublishToStartedRequest) (*MoveFromPublishToStartedResponse, error) {
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.MoveFromPublishToStarted")
	defer span.End()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "MoveFromPublishToStarted"),
		slog.String("client_id", req.DriverID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

    cfg := config.GetAppConfig()

    tripStartAllowedResp, err := s.contractClient.CheckService(
        ctx, cfg.Company.CompanyID, entity.ServiceTripStart,
    )
    if err != nil {
        logger.Error("failed to check contract service", slog.Any("error", err))
        return nil, fmt.Errorf("%w: %s", tripErrors.ErrTripStartIsNotAllowed, err.Error())
    }
    if !tripStartAllowedResp.Allowed {
        return nil, fmt.Errorf("%w: %s", tripErrors.ErrTripStartIsNotAllowed, tripStartAllowedResp.Reason)
    }

	started := time.Now()
	result := "success"

	defer func() {
		s.metrics.TripPublishTotal.WithLabelValues(result).Inc()
		s.metrics.TripPublishDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	logger.Info("move from publish to started started")

	domainReq := domain.MoveFromPublishToStartedRequest{
		TripID:    req.TripID,
		DriverID:  req.DriverID,
		OldStatus: req.OldStatus,
		NewStatus: req.NewStatus,
	}

	domainResp, err := s.tripDomain.MoveFromPublishToStarted(ctx, domainReq)
	if err != nil {
		logger.Error("failed to move from publish to started", slog.Any("error", err))
		result = "error"
		return nil, err
	}

	logger.Info("move from publish to started completed", slog.String("new_status", domainResp.Status))

	return &MoveFromPublishToStartedResponse{
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