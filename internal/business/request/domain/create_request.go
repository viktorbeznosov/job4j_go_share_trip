package domain

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type CreateRequestInput struct {
	PassengerID   uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

func (d *RequestDomain) Create(ctx context.Context, req CreateRequestInput) (*RequestResponse, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("domain", "RequestDomain"),
		slog.String("operation", "CreateRequest"),
		slog.String("client_id", req.PassengerID.String()),
	)

	request, err := entity.NewRequest(
		req.PassengerID,
		req.FromPoint,
		req.ToPoint,
		req.DepartureTime,
		req.Seats,
	)
	if err != nil {
		logger.Error("failed to create request entity", slog.Any("error", err))
		return nil, err
	}

	err = storage.TxWithoutResult(ctx, d.requestRepository.GetDB(), func(tx pgx.Tx) error {
		if err := d.requestRepository.CreateTx(ctx, tx, request); err != nil {
			logger.Error("failed to create request in transaction", slog.Any("error", err))
			return err
		}

		if err := d.requestRepository.CreateHistoryTx(ctx, tx, request.ID, nil, &request.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return err
		}

		if err := d.saveOutboxTx(ctx, tx, outbox.EventTypeRequestCreated, events.EventTypeRequestCreated, request, nil); err != nil {
			logger.Error("failed to save outbox event", slog.Any("error", err))
			return err
		}

		return nil
	})
	if err != nil {
		logger.Error("failed to create request", slog.Any("error", err))
		return nil, err
	}

	logger.Info("create request completed", slog.String("request_id", request.ID.String()))

	return newRequestResponse(request), nil
}
