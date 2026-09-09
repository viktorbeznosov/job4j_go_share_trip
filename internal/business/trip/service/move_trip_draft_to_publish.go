package service

import (
	"context"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/observability/logctx"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
)

type MoveFromDraftToPublishRequest struct {
	TripID    uuid.UUID
	DriverID  uuid.UUID
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
	ctx, span := otel.Tracer("TripService").Start(ctx, "TripService.Update")
	defer span.End()

	started := time.Now()
	result := "success"

	defer func() {
		s.metrics.TripPublishTotal.WithLabelValues(result).Inc()
		s.metrics.TripPublishDuration.WithLabelValues(result).
			Observe(time.Since(started).Seconds())
	}()

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "ModeTripFromDraftToPublished"),
		slog.String("client_id", req.DriverID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	logger.Info("update trip from draft to publish started")

	domainReq := domain.MoveFromDraftToPublishRequest{
		TripID:    req.TripID,
		DriverID:  req.DriverID,
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