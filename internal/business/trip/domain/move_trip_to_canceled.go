package domain

import (
	"context"
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

type MoveTripToCanceledRequest struct {
	TripID    uuid.UUID
	ClientID  uuid.UUID
	CompanyID uuid.UUID
}

type MoveTripToCanceledResponse struct {
	ID              uuid.UUID
	DriverID        uuid.UUID
	FromPoint       string
	ToPoint         string
	DepartureTime   time.Time
	Seats           int
	Status          string
	CreatedAt       time.Time
	AlreadyCanceled bool
}

func (d *TripDomain) MoveToCanceled(
	ctx context.Context,
	req MoveTripToCanceledRequest,
) (*MoveTripToCanceledResponse, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("domain", "TripDomain"),
		slog.String("operation", "MoveToCanceled"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	trip, err := d.tripRepository.GetByTripID(ctx, req.TripID)
	if err != nil {
		logger.Error("failed to get trip", slog.Any("error", err))
		return nil, err
	}

	if trip.Status == entity.StatusCanceled {
		logger.Info("trip already canceled, skipping update")
		return newMoveTripToCanceledResponse(trip, true), nil
	}

	if req.ClientID != trip.DriverID {
		logger.Warn("Forbidden: client is not driver",
			slog.String("client_id", req.ClientID.String()),
			slog.String("driver_id", trip.DriverID.String()),
		)
		return nil, fmt.Errorf("%w: client %s is not the owner of trip %s",
			tripErrors.ErrDriverNotOwner, req.ClientID, trip.ID)
	}

	if trip.Status == entity.StatusStarted {
		return nil, fmt.Errorf("%w: trip status is %s",
			tripErrors.ErrTripCancelNotAllowed, trip.Status)
	}

	alreadyCanceled := false
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

		if lockedTrip.Status == entity.StatusCanceled {
			alreadyCanceled = true
			return &lockedTrip, nil
		}

		if lockedTrip.Status == entity.StatusStarted {
			return nil, fmt.Errorf("%w: trip status is %s",
				tripErrors.ErrTripCancelNotAllowed, lockedTrip.Status)
		}

		oldStatus := lockedTrip.Status
		lockedTrip.Status = entity.StatusCanceled

		if err := d.tripRepository.UpdateTx(ctx, tx, &lockedTrip); err != nil {
			logger.Error("failed to update trip in transaction", slog.Any("error", err))
			return nil, err
		}

		if err := d.tripRepository.CreateHistoryTx(ctx, tx, lockedTrip.ID, &oldStatus, &lockedTrip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return nil, err
		}

		attached, err := d.listAttachedRequestsTx(ctx, tx, lockedTrip.ID)
		if err != nil {
			logger.Error("failed to list attached requests", slog.Any("error", err))
			return nil, err
		}

		passengerIDs, requestIDs, err := d.detachRequestsToPublishedTx(ctx, tx, attached)
		if err != nil {
			logger.Error("failed to detach attached requests", slog.Any("error", err))
			return nil, err
		}

		if err := d.saveTripOutboxTx(
			ctx,
			tx,
			lockedTrip.ID,
			outbox.EventTypeTripCanceled,
			events.EventTypeTripCanceled,
			func(eventID uuid.UUID, occurredAt time.Time, meta events.EventMetadata) (any, error) {
				return events.NewTripCanceled(
					eventID,
					lockedTrip.ID,
					lockedTrip.DriverID,
					req.CompanyID,
					occurredAt,
					meta,
					passengerIDs,
					requestIDs,
				), nil
			},
		); err != nil {
			logger.Error("failed to save outbox event", slog.Any("error", err))
			return nil, err
		}

		return &lockedTrip, nil
	})
	if err != nil {
		logger.Error("failed to move trip to canceled", slog.Any("error", err))
		return nil, err
	}

	logger.Info("trip moved to canceled successfully",
		slog.String("new_status", string(updatedTrip.Status)),
	)

	return newMoveTripToCanceledResponse(updatedTrip, alreadyCanceled), nil
}

func newMoveTripToCanceledResponse(
	trip *entity.Trip,
	alreadyCanceled bool,
) *MoveTripToCanceledResponse {
	return &MoveTripToCanceledResponse{
		ID:              trip.ID,
		DriverID:        trip.DriverID,
		FromPoint:       trip.FromPoint,
		ToPoint:         trip.ToPoint,
		DepartureTime:   trip.DepartureTime,
		Seats:           trip.Seats,
		Status:          string(trip.Status),
		CreatedAt:       trip.CreatedAt,
		AlreadyCanceled: alreadyCanceled,
	}
}
