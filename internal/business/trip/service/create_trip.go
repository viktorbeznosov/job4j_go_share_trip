package service

import (
	"context"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/observability/logctx"

	"github.com/google/uuid"
)

type CreateTripRequest struct {
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

type CreateTripResponse struct {
	ID            uuid.UUID
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        string
	CreatedAt     time.Time
}

func (s *TripService) Create(ctx context.Context, req CreateTripRequest) (*CreateTripResponse, error) {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("service", "TripService"),
		slog.String("operation", "CreateTrip"),
		slog.String("client_id", req.DriverID.String()),
	)

	logger.Info("create trip started")

	domainReq := domain.CreateTripRequest{
		DriverID:      req.DriverID,
		FromPoint:     req.FromPoint,
		ToPoint:       req.ToPoint,
		DepartureTime: req.DepartureTime,
		Seats:         req.Seats,
	}

	domainResp, err := s.tripDomain.Create(ctx, domainReq)
	if err != nil {
		logger.Error("failed to create trip in domain", slog.Any("error", err))
		result = "error"
		s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
		s.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
		return nil, err
	}

	logger.Info("create trip completed", slog.String("trip_id", domainResp.ID.String()))

	s.metrics.TripCreateTotal.WithLabelValues(result).Inc()
	s.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())

	return &CreateTripResponse{
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