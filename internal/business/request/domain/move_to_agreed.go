package domain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/entity"
	tripEntity "job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type MoveToAgreedInput struct {
	RequestID   uuid.UUID
	PassengerID uuid.UUID
	TripID      uuid.UUID
}

func (d *RequestDomain) MoveToAgreed(ctx context.Context, req MoveToAgreedInput) (*RequestResponse, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("domain", "RequestDomain"),
		slog.String("operation", "MoveToAgreed"),
		slog.String("client_id", req.PassengerID.String()),
		slog.String("request_id", req.RequestID.String()),
		slog.String("trip_id", req.TripID.String()),
	)

	request, err := d.requestRepository.GetByID(ctx, req.RequestID)
	if err != nil {
		return nil, err
	}

	if request.PassengerID != req.PassengerID {
		return nil, fmt.Errorf("%w: client %s is not the owner of request %s",
			tripErrors.ErrPassengerNotOwner, req.PassengerID, request.ID)
	}

	if request.Status != entity.StatusPublished && request.Status != entity.StatusPreAgreed {
		return nil, fmt.Errorf("%w: invalid request status: expected %s or %s, got %s",
			tripErrors.ErrRequestNotPublished, entity.StatusPublished, entity.StatusPreAgreed, request.Status)
	}

	updated, err := storage.Tx(ctx, d.requestRepository.GetDB(), func(tx pgx.Tx) (*entity.Request, error) {
		trip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, req.TripID)
		if err != nil {
			return nil, err
		}

		if trip.Status != tripEntity.StatusPublished {
			return nil, fmt.Errorf("%w: invalid trip status: expected %s, got %s",
				tripErrors.ErrTripNotPublished, tripEntity.StatusPublished, trip.Status)
		}

		locked, err := d.requestRepository.GetForUpdateByIDWithTX(ctx, tx, req.RequestID)
		if err != nil {
			return nil, err
		}

		if locked.PassengerID != req.PassengerID {
			return nil, fmt.Errorf("%w: client %s is not the owner of request %s",
				tripErrors.ErrPassengerNotOwner, req.PassengerID, locked.ID)
		}

		if locked.Status != entity.StatusPublished && locked.Status != entity.StatusPreAgreed {
			return nil, fmt.Errorf("%w: invalid request status: expected %s or %s, got %s",
				tripErrors.ErrRequestNotPublished, entity.StatusPublished, entity.StatusPreAgreed, locked.Status)
		}

		if err := d.ensureCapacityTx(ctx, tx, trip.Seats, trip.ID, &locked); err != nil {
			return nil, err
		}

		oldStatus := locked.Status
		locked.Status = entity.StatusPreAgreed
		tripID := trip.ID
		locked.TripID = &tripID

		if err := d.requestRepository.UpdateTx(ctx, tx, &locked); err != nil {
			return nil, err
		}
		if err := d.requestRepository.CreateHistoryTx(ctx, tx, locked.ID, &oldStatus, &locked.Status); err != nil {
			return nil, err
		}
		if err := d.saveOutboxTx(ctx, tx, outbox.EventTypeRequestPreconfirmed, events.EventTypeRequestPreconfirmed, &locked, &trip.DriverID); err != nil {
			return nil, err
		}

		return &locked, nil
	})
	if err != nil {
		logger.Error("failed to attach request to trip", slog.Any("error", err))
		return nil, err
	}

	return newRequestResponse(updated), nil
}
