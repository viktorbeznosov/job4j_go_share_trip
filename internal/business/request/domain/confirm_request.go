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

type ConfirmRequestInput struct {
	RequestID uuid.UUID
	ClientID  uuid.UUID
	Confirm   bool
}

type ConfirmRequestResult struct {
	Request       *RequestResponse
	AlreadyAgreed bool
}

func (d *RequestDomain) Confirm(ctx context.Context, req ConfirmRequestInput) (*ConfirmRequestResult, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("domain", "RequestDomain"),
		slog.String("operation", "ConfirmRequest"),
		slog.String("client_id", req.ClientID.String()),
		slog.String("request_id", req.RequestID.String()),
		slog.Bool("confirm", req.Confirm),
	)

	request, err := d.requestRepository.GetByID(ctx, req.RequestID)
	if err != nil {
		return nil, err
	}

	if req.Confirm {
		return d.confirmByDriver(ctx, logger, req, request)
	}

	return d.rejectRequest(ctx, logger, req, request)
}

func (d *RequestDomain) confirmByDriver(
	ctx context.Context,
	logger *slog.Logger,
	req ConfirmRequestInput,
	request *entity.Request,
) (*ConfirmRequestResult, error) {
	if request.Status == entity.StatusAgreed {
		return &ConfirmRequestResult{Request: newRequestResponse(request), AlreadyAgreed: true}, nil
	}

	if request.Status != entity.StatusPreAgreed || request.TripID == nil {
		return nil, fmt.Errorf("%w: invalid request status: expected %s, got %s",
			tripErrors.ErrRequestNotPreAgreed, entity.StatusPreAgreed, request.Status)
	}

	alreadyAgreed := false
	updated, err := storage.Tx(ctx, d.requestRepository.GetDB(), func(tx pgx.Tx) (*entity.Request, error) {
		trip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, *request.TripID)
		if err != nil {
			return nil, err
		}

		if trip.DriverID != req.ClientID {
			return nil, fmt.Errorf("%w: client %s is not the owner of trip %s",
				tripErrors.ErrDriverNotOwner, req.ClientID, trip.ID)
		}

		if trip.Status != tripEntity.StatusPublished {
			return nil, fmt.Errorf("%w: invalid trip status: expected %s, got %s",
				tripErrors.ErrTripNotPublished, tripEntity.StatusPublished, trip.Status)
		}

		locked, err := d.requestRepository.GetForUpdateByIDWithTX(ctx, tx, req.RequestID)
		if err != nil {
			return nil, err
		}

		if locked.Status == entity.StatusAgreed {
			alreadyAgreed = true
			return &locked, nil
		}

		if locked.Status != entity.StatusPreAgreed || locked.TripID == nil {
			return nil, fmt.Errorf("%w: invalid request status: expected %s, got %s",
				tripErrors.ErrRequestNotPreAgreed, entity.StatusPreAgreed, locked.Status)
		}

		if err := d.ensureCapacityTx(ctx, tx, trip.Seats, trip.ID, &locked); err != nil {
			return nil, err
		}

		oldStatus := locked.Status
		locked.Status = entity.StatusAgreed

		if err := d.requestRepository.UpdateTx(ctx, tx, &locked); err != nil {
			return nil, err
		}
		if err := d.requestRepository.CreateHistoryTx(ctx, tx, locked.ID, &oldStatus, &locked.Status); err != nil {
			return nil, err
		}
		if err := d.saveOutboxTx(ctx, tx, outbox.EventTypeRequestConfirmed, events.EventTypeRequestConfirmed, &locked, &trip.DriverID); err != nil {
			return nil, err
		}

		return &locked, nil
	})
	if err != nil {
		logger.Error("failed to confirm request", slog.Any("error", err))
		return nil, err
	}

	return &ConfirmRequestResult{
		Request:       newRequestResponse(updated),
		AlreadyAgreed: alreadyAgreed,
	}, nil
}

func (d *RequestDomain) rejectRequest(
	ctx context.Context,
	logger *slog.Logger,
	req ConfirmRequestInput,
	request *entity.Request,
) (*ConfirmRequestResult, error) {
	if request.Status != entity.StatusPreAgreed {
		return nil, fmt.Errorf("%w: invalid request status: expected %s, got %s",
			tripErrors.ErrRequestNotPreAgreed, entity.StatusPreAgreed, request.Status)
	}

	isPassenger := request.PassengerID == req.ClientID
	if !isPassenger && request.TripID == nil {
		return nil, tripErrors.ErrRequestConfirmForbidden
	}

	updated, err := storage.Tx(ctx, d.requestRepository.GetDB(), func(tx pgx.Tx) (*entity.Request, error) {
		locked, err := d.requestRepository.GetForUpdateByIDWithTX(ctx, tx, req.RequestID)
		if err != nil {
			return nil, err
		}

		if locked.Status != entity.StatusPreAgreed {
			return nil, fmt.Errorf("%w: invalid request status: expected %s, got %s",
				tripErrors.ErrRequestNotPreAgreed, entity.StatusPreAgreed, locked.Status)
		}

		allowed := locked.PassengerID == req.ClientID
		if !allowed {
			if locked.TripID == nil {
				return nil, tripErrors.ErrRequestConfirmForbidden
			}
			trip, err := d.tripRepository.GetForUpdateByIDWithTX(ctx, tx, *locked.TripID)
			if err != nil {
				return nil, err
			}
			if trip.DriverID != req.ClientID {
				return nil, tripErrors.ErrRequestConfirmForbidden
			}
		}

		oldStatus := locked.Status
		locked.Status = entity.StatusPublished
		locked.TripID = nil

		if err := d.requestRepository.UpdateTx(ctx, tx, &locked); err != nil {
			return nil, err
		}
		if err := d.requestRepository.CreateHistoryTx(ctx, tx, locked.ID, &oldStatus, &locked.Status); err != nil {
			return nil, err
		}
		if err := d.saveOutboxTx(ctx, tx, outbox.EventTypeRequestRejected, events.EventTypeRequestRejected, &locked, nil); err != nil {
			return nil, err
		}

		return &locked, nil
	})
	if err != nil {
		logger.Error("failed to reject request", slog.Any("error", err))
		return nil, err
	}

	return &ConfirmRequestResult{Request: newRequestResponse(updated)}, nil
}
