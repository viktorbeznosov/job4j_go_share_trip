package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"job4j_go_share_trip/config"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type MoveFromDraftToPublishRequest struct {
	Trip      GetTripResponse
	ClientID  uuid.UUID
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

	cfg := config.GetAppConfig()

	tripPublishAllowedResp, err := s.contractClient.CheckService(
		ctx, cfg.Company.CompanyID, entity.ServiceTripPublish,
	)
	if err != nil {
		logger.Error("failed to check contract service", slog.Any("error", err))
		return nil, fmt.Errorf("%w: %s", tripErrors.ErrTripPublishIsNotAllowed, err.Error())
	}
	if !tripPublishAllowedResp.Allowed {
		return nil, fmt.Errorf("%w: %s", tripErrors.ErrTripPublishIsNotAllowed, tripPublishAllowedResp.Reason)
	}

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

	if s.tripPublisher != nil {
		event := events.TripPublished{
			EventID:    uuid.New().String(),
			EventType:  events.EventTypeTripPublished,
			TripID:     domainResp.ID.String(),
			DriverID:   domainResp.DriverID.String(),
			CompanyID:  cfg.Company.CompanyID,
			OccurredAt: time.Now(),
		}
		if err := s.tripPublisher.PublishTripPublished(ctx, event); err != nil {
			logger.Error("failed to publish trip_published event", slog.Any("error", err))
			result = "error"
			return nil, fmt.Errorf("failed to publish trip_published event: %w", err)
		}
	}

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
