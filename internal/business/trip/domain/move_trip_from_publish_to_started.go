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
	TripID   uuid.UUID
	ClientID uuid.UUID
}

type MoveFromPublishToStartedResponse struct {
	ID             uuid.UUID
	DriverID       uuid.UUID
	FromPoint      string
	ToPoint        string
	DepartureTime  time.Time
	Seats          int
	Status         string
	CreatedAt      time.Time
	AlreadyStarted bool
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
		slog.String("trip_id", req.TripID.String()),
	)

	trip, err := d.tripRepository.GetByTripID(ctx, req.TripID)
	if err != nil {
		logger.Error("failed to get trip", slog.Any("error", err))
		return nil, err
	}

	if trip.Status == entity.StatusStarted {
		logger.Info("trip already started, skipping update")
		return newMoveFromPublishToStartedResponse(trip, true), nil
	}

	if trip.Status != entity.StatusPublished {
		return nil, fmt.Errorf(
			"%w: invalid trip status: expected %s, got %s",
			tripErrors.ErrTripNotPublished,
			entity.StatusPublished,
			trip.Status,
		)
	}

	if req.ClientID != trip.DriverID {
		logger.Warn("Forbidden: client is not driver",
			slog.String("client_id", req.ClientID.String()),
			slog.String("driver_id", trip.DriverID.String()),
		)
		return nil, fmt.Errorf("%w: client %s is not the owner of trip %s",
			tripErrors.ErrDriverNotOwner, req.ClientID, trip.ID)
	}

	alreadyStarted := false
	updatedTrip, err := storage.Tx(ctx, d.tripRepository.GetDB(), func(tx pgx.Tx) (*entity.Trip, error) {
		lockedTrip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, req.TripID)
		if err != nil {
			logger.Error("failed to get trip for update", slog.Any("error", err))
			return nil, err
		}

		if lockedTrip.DriverID != req.ClientID {
			return nil, fmt.Errorf("%w: driver %s is not the owner of trip %s",
				tripErrors.ErrDriverNotOwner, req.ClientID, lockedTrip.ID)
		}

		if lockedTrip.Status == entity.StatusStarted {
			alreadyStarted = true
			return &lockedTrip, nil
		}

		if lockedTrip.Status != entity.StatusPublished {
			return nil, fmt.Errorf(
				"%w: invalid trip status: expected %s, got %s",
				tripErrors.ErrTripNotPublished,
				entity.StatusPublished,
				lockedTrip.Status,
			)
		}

		oldStatus := lockedTrip.Status
		lockedTrip.Status = entity.StatusStarted

		if err := d.tripRepository.UpdateTx(ctx, tx, &lockedTrip); err != nil {
			logger.Error("failed to update trip in transaction", slog.Any("error", err))
			return nil, err
		}

		if err := d.tripRepository.CreateHistoryTx(ctx, tx, lockedTrip.ID, &oldStatus, &lockedTrip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return nil, err
		}

		payload, err := json.Marshal(lockedTrip)
		if err != nil {
			return nil, err
		}

		event := outbox.Event{
			ID:          uuid.New(),
			EventName:   outbox.TripStarted,
			AggregateID: lockedTrip.ID,
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

		return &lockedTrip, nil
	})

	if err != nil {
		logger.Error("failed to move from publish to started", slog.Any("error", err))
		return nil, err
	}

	logger.Info("trip moved from publish to started successfully",
		slog.String("new_status", string(updatedTrip.Status)),
	)

	return newMoveFromPublishToStartedResponse(updatedTrip, alreadyStarted), nil
}

func newMoveFromPublishToStartedResponse(
	trip *entity.Trip,
	alreadyStarted bool,
) *MoveFromPublishToStartedResponse {
	return &MoveFromPublishToStartedResponse{
		ID:             trip.ID,
		DriverID:       trip.DriverID,
		FromPoint:      trip.FromPoint,
		ToPoint:        trip.ToPoint,
		DepartureTime:  trip.DepartureTime,
		Seats:          trip.Seats,
		Status:         string(trip.Status),
		CreatedAt:      trip.CreatedAt,
		AlreadyStarted: alreadyStarted,
	}
}
