package domain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type MoveToPublishInput struct {
	RequestID   uuid.UUID
	PassengerID uuid.UUID
}

type MoveToPublishResult struct {
	Request          *RequestResponse
	AlreadyPublished bool
}

func (d *RequestDomain) MoveToPublish(ctx context.Context, req MoveToPublishInput) (*MoveToPublishResult, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("domain", "RequestDomain"),
		slog.String("operation", "MoveToPublish"),
		slog.String("client_id", req.PassengerID.String()),
		slog.String("request_id", req.RequestID.String()),
	)

	request, err := d.requestRepository.GetByID(ctx, req.RequestID)
	if err != nil {
		return nil, err
	}

	if request.PassengerID != req.PassengerID {
		return nil, fmt.Errorf("%w: client %s is not the owner of request %s",
			tripErrors.ErrPassengerNotOwner, req.PassengerID, request.ID)
	}

	if request.Status == entity.StatusPublished {
		return &MoveToPublishResult{Request: newRequestResponse(request), AlreadyPublished: true}, nil
	}

	if request.Status != entity.StatusDraft {
		return nil, fmt.Errorf("%w: invalid request status: expected %s, got %s",
			tripErrors.ErrRequestNotDraft, entity.StatusDraft, request.Status)
	}

	alreadyPublished := false
	updated, err := storage.Tx(ctx, d.requestRepository.GetDB(), func(tx pgx.Tx) (*entity.Request, error) {
		locked, err := d.requestRepository.GetForUpdateByIDWithTX(ctx, tx, req.RequestID)
		if err != nil {
			return nil, err
		}

		if locked.PassengerID != req.PassengerID {
			return nil, fmt.Errorf("%w: client %s is not the owner of request %s",
				tripErrors.ErrPassengerNotOwner, req.PassengerID, locked.ID)
		}

		if locked.Status == entity.StatusPublished {
			alreadyPublished = true
			return &locked, nil
		}

		if locked.Status != entity.StatusDraft {
			return nil, fmt.Errorf("%w: invalid request status: expected %s, got %s",
				tripErrors.ErrRequestNotDraft, entity.StatusDraft, locked.Status)
		}

		oldStatus := locked.Status
		locked.Status = entity.StatusPublished

		if err := d.requestRepository.UpdateTx(ctx, tx, &locked); err != nil {
			return nil, err
		}
		if err := d.requestRepository.CreateHistoryTx(ctx, tx, locked.ID, &oldStatus, &locked.Status); err != nil {
			return nil, err
		}
		if err := d.saveOutboxTx(ctx, tx, outbox.EventTypeRequestPublished, events.EventTypeRequestPublished, &locked, nil); err != nil {
			return nil, err
		}

		return &locked, nil
	})
	if err != nil {
		logger.Error("failed to publish request", slog.Any("error", err))
		return nil, err
	}

	return &MoveToPublishResult{
		Request:          newRequestResponse(updated),
		AlreadyPublished: alreadyPublished,
	}, nil
}
