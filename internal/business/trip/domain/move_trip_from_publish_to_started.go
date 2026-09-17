package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type MoveFromPublishToStartedRequest struct {
	Trip      GetTripResponse
	ClientID  uuid.UUID
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

func (d *TripDomain) MoveFromPublishToStarted(
	ctx context.Context,
	req MoveFromPublishToStartedRequest,
) (*MoveFromPublishToStartedResponse, error) {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("domain", "TripDomain"),
		slog.String("operation", "MoveFromPublishToStarted"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.Trip.ID.String()),
	)

	if req.ClientID != req.Trip.DriverID {
		logger.Warn("Forbidden: client is not driver",
			slog.String("client_id", req.ClientID.String()),
			slog.String("driver_id", req.Trip.DriverID.String()),
		)
		return nil, fmt.Errorf("%w: client %s is not the owner of trip %s",
			tripErrors.ErrDriverNotOwner, req.ClientID, req.Trip.ID)
	}

	if req.OldStatus != string(entity.StatusPublished) {
		return nil, fmt.Errorf("%w: expected %s, got %s",
			tripErrors.ErrTripNotPublished, entity.StatusPublished, req.OldStatus)
	}

	if req.NewStatus != string(entity.StatusStarted) {
		return nil, fmt.Errorf("%w: expected %s, got %s",
			tripErrors.ErrInvalidStatusTransition, entity.StatusStarted, req.NewStatus)
	}

	trip, err := storage.Tx(ctx, d.tripRepository.GetDB(), func(tx pgx.Tx) (*entity.Trip, error) {
		trip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, req.Trip.ID)
		if err != nil {
			logger.Error("failed to get trip for update", slog.Any("error", err))
			return nil, err
		}

		if trip.DriverID != req.ClientID {
			return nil, fmt.Errorf("%w: driver %s is not the owner of trip %s",
				tripErrors.ErrDriverNotOwner, req.ClientID, trip.ID)
		}

		if string(trip.Status) != req.OldStatus {
			return nil, fmt.Errorf("%w: expected %s, got %s",
				tripErrors.ErrTripNotPublished, req.OldStatus, trip.Status)
		}

		oldStatus := trip.Status
		trip.Status = entity.Status(req.NewStatus)

		if err := d.tripRepository.UpdateTx(ctx, tx, &trip); err != nil {
			logger.Error("failed to update trip in transaction", slog.Any("error", err))
			return nil, err
		}

		if err := d.tripRepository.CreateHistoryTx(ctx, tx, trip.ID, &oldStatus, &trip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return nil, err
		}

		payload, err := json.Marshal(trip)
		if err != nil {
			return nil, err
		}

		event := outbox.Event{
			ID:          uuid.New(),
			EventName:   outbox.TripStarted,
			AggregateID: trip.ID,
			Payload:     payload,
			CreatedAt:   time.Now(),
		}

		if err := d.eventRepository.SaveTx(ctx, tx, &event); err != nil {
			logger.Error("failed to save outbox event", slog.Any("error", err))
			result = "error"
			d.metrics.TripCreateTotal.WithLabelValues(result).Inc()
			d.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
			return nil, fmt.Errorf("failed to save outbox event: %w", err)
		}

		return &trip, nil
	})

	if err != nil {
		logger.Error("failed to move from publish to started", slog.Any("error", err))
		return nil, err
	}

	logger.Info("trip moved from publish to started successfully",
		slog.String("new_status", string(trip.Status)),
	)

	return &MoveFromPublishToStartedResponse{
		ID:            trip.ID,
		DriverID:      trip.DriverID,
		FromPoint:     trip.FromPoint,
		ToPoint:       trip.ToPoint,
		DepartureTime: trip.DepartureTime,
		Seats:         trip.Seats,
		Status:        string(trip.Status),
		CreatedAt:     trip.CreatedAt,
	}, nil
}