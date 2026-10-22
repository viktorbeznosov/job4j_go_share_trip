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
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
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

func (d *TripDomain) ApproveTrip(
	ctx context.Context,
	req ApproveTripRequest,
) (*ApproveTripResponse, error) {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("domain", "TripDomain"),
		slog.String("operation", "ApproveTrip"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	trip, err := d.tripRepository.GetByTripID(ctx, req.TripID)
	if err != nil {
		logger.Error("failed to get trip", slog.Any("error", err))
		return nil, err
	}

	if trip.Status == entity.StatusPublished {
		logger.Info("trip already published, skipping update")
		return newApproveTripResponse(trip, true), nil
	}

	if trip.Status != entity.StatusPreapproved {
		return nil, fmt.Errorf(
			"%w: invalid trip status: expected %s, got %s",
			tripErrors.ErrTripNotPreapproved,
			entity.StatusPreapproved,
			trip.Status,
		)
	}

	alreadyPublished := false
	updatedTrip, err := storage.Tx(ctx, d.tripRepository.GetDB(), func(tx pgx.Tx) (*entity.Trip, error) {
		lockedTrip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, req.TripID)
		if err != nil {
			logger.Error("failed to get trip for update", slog.Any("error", err))
			return nil, err
		}

		if lockedTrip.Status == entity.StatusPublished {
			alreadyPublished = true
			return &lockedTrip, nil
		}

		if lockedTrip.Status != entity.StatusPreapproved {
			return nil, fmt.Errorf(
				"%w: invalid trip status: expected %s, got %s",
				tripErrors.ErrTripNotPreapproved,
				entity.StatusPreapproved,
				lockedTrip.Status,
			)
		}

		oldStatus := lockedTrip.Status
		lockedTrip.Status = entity.StatusPublished

		if err := d.tripRepository.UpdateTx(ctx, tx, &lockedTrip); err != nil {
			logger.Error("failed to update trip in transaction", slog.Any("error", err))
			return nil, err
		}

		if err := d.tripRepository.CreateHistoryTx(ctx, tx, lockedTrip.ID, &oldStatus, &lockedTrip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return nil, err
		}

		event := outbox.NewPendingEvent(outbox.EventTypeTripPublished, lockedTrip.ID, nil)
		meta := events.MetadataFromContext(
			ctx,
			event.ID.String(),
			events.EventTypeTripPublished,
			event.CreatedAt,
		)
		metadataJSON, err := json.Marshal(meta)
		if err != nil {
			return nil, err
		}
		event.Metadata = metadataJSON

		payload, err := json.Marshal(events.NewTripPublished(
			event.ID,
			lockedTrip.ID,
			lockedTrip.DriverID,
			req.CompanyID,
			event.CreatedAt,
			meta,
		))
		if err != nil {
			return nil, err
		}
		event.Payload = payload

		if err := d.eventRepository.SaveTx(ctx, tx, &event); err != nil {
			logger.Error("failed to save outbox event", slog.Any("error", err))
			result = "error"
			d.metrics.TripCreateTotal.WithLabelValues(result).Inc()
			d.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
			return nil, fmt.Errorf("failed to save outbox event: %w", err)
		}

		logger.Info("outbox event created",
			slog.String("operation", "OutboxInsert"),
			slog.String("event_id", event.ID.String()),
			slog.String("event_type", string(outbox.EventTypeTripPublished)),
			slog.String("correlation_id", meta.CorrelationID),
			slog.String("causation_id", meta.CausationID),
			slog.String("result", "created"),
		)

		return &lockedTrip, nil
	})

	if err != nil {
		logger.Error("failed to approve trip", slog.Any("error", err))
		return nil, err
	}

	logger.Info("trip approved successfully",
		slog.String("new_status", string(updatedTrip.Status)),
	)

	return newApproveTripResponse(updatedTrip, alreadyPublished), nil
}

func newApproveTripResponse(
	trip *entity.Trip,
	alreadyPublished bool,
) *ApproveTripResponse {
	return &ApproveTripResponse{
		ID:               trip.ID,
		DriverID:         trip.DriverID,
		FromPoint:        trip.FromPoint,
		ToPoint:          trip.ToPoint,
		DepartureTime:    trip.DepartureTime,
		Seats:            trip.Seats,
		Status:           string(trip.Status),
		CreatedAt:        trip.CreatedAt,
		AlreadyPublished: alreadyPublished,
	}
}
