package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
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

func (d *TripDomain) MoveFromPublishToStarted(ctx context.Context, req MoveFromPublishToStartedRequest) (*MoveFromPublishToStartedResponse, error) {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("domain", "TripDomain"),
		slog.String("operation", "MoveFromPublishToStarted"),
		slog.String("client_id", req.DriverID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	if req.OldStatus != string(entity.StatusPublished) {
		return nil, fmt.Errorf("invalid old status: expected %s, got %s", entity.StatusPublished, req.OldStatus)
	}

	if req.NewStatus != string(entity.StatusStarted) {
		return nil, fmt.Errorf("invalid new status: expected %s, got %s", entity.StatusStarted, req.NewStatus)
	}

	trip, err := storage.Tx(ctx, d.tripRepository.GetDB(), func(tx pgx.Tx) (*entity.Trip, error) {
		trip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, req.TripID)
		if err != nil {
			logger.Error("failed to get trip for update", slog.Any("error", err))
			return nil, err
		}

		if string(trip.Status) != req.OldStatus {
			return nil, fmt.Errorf("invalid trip status: expected %s, got %s", req.OldStatus, trip.Status)
		}

		if trip.DriverID != req.DriverID {
			return nil, fmt.Errorf("driver %s is not the owner of trip %s", req.DriverID, req.TripID)
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

	logger.Info("trip moved from publish to started successfully", slog.String("new_status", string(trip.Status)))

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