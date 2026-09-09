// internal/business/trip/domain/move_trip_draft_to_publish.go
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

func (d *TripDomain) MoveFromDraftToPublish(ctx context.Context, req MoveFromDraftToPublishRequest) (*MoveFromDraftToPublishResponse, error) {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("domain", "TripDomain"),
		slog.String("operation", "MoveFromDraftToPublish"),
		slog.String("client_id", req.DriverID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	if req.OldStatus != string(entity.StatusDraft) {
		return nil, fmt.Errorf("invalid old status: expected %s, got %s", entity.StatusDraft, req.OldStatus)
	}

	if req.NewStatus != string(entity.StatusPublished) {
		return nil, fmt.Errorf("invalid new status: expected %s, got %s", entity.StatusPublished, req.NewStatus)
	}

	trip, err := storage.Tx(ctx, d.tripRepository.GetDB(), func(tx pgx.Tx) (*entity.Trip, error) {
		trip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, req.TripID)
		if err != nil {
			logger.Error("failed to get trip for update", slog.Any("error", err))
			return nil, err
		}

		if trip.DriverID != req.DriverID {
			return nil, fmt.Errorf("driver %s is not the owner of trip %s", req.DriverID, req.TripID)
		}

		if trip.Status == entity.StatusPublished {
			logger.Info("trip already published, skipping update")
			return &trip, nil
		}

		if trip.Status != entity.StatusDraft {
			return nil, fmt.Errorf("invalid trip status: expected %s, got %s", entity.StatusDraft, trip.Status)
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
			EventName:   outbox.TripPublished,
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
		logger.Error("failed to move from draft to publish", slog.Any("error", err))
		return nil, err
	}

	logger.Info("trip moved from draft to publish successfully", slog.String("new_status", string(trip.Status)))

	return &MoveFromDraftToPublishResponse{
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