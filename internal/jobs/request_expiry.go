package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/business/request/entity"
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type RequestExpiry struct {
	requests *requestRepo.RequestRepository
	outbox   *outbox.EventRepository
}

func NewRequestExpiry(
	requests *requestRepo.RequestRepository,
	eventsRepo *outbox.EventRepository,
) *RequestExpiry {
	return &RequestExpiry{
		requests: requests,
		outbox:   eventsRepo,
	}
}

func (e *RequestExpiry) Tick(ctx context.Context) error {
	logger := logctx.Logger(ctx).With(slog.String("job", "RequestExpiry"))
	err := storage.TxWithoutResult(ctx, e.requests.GetDB(), func(tx pgx.Tx) error {
		items, err := e.requests.ListExpiredForUpdateTx(ctx, tx)
		if err != nil {
			return err
		}

		for i := range items {
			if err := e.expire(ctx, tx, &items[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("request expiry tick failed", slog.Any("error", err))
	}
	return err
}

func (e *RequestExpiry) expire(ctx context.Context, tx pgx.Tx, request *entity.Request) error {
	oldStatus := request.Status
	request.Status = entity.StatusCanceled
	request.TripID = nil

	if err := e.requests.UpdateTx(ctx, tx, request); err != nil {
		return err
	}
	if err := e.requests.CreateHistoryTx(ctx, tx, request.ID, &oldStatus, &request.Status); err != nil {
		return err
	}

	event := outbox.NewPendingAggregateEvent(
		outbox.AggregateTypeRequest,
		outbox.EventTypeRequestExpired,
		request.ID,
		nil,
	)
	meta := events.MetadataFromContext(ctx, event.ID.String(), events.EventTypeRequestExpired, event.CreatedAt)
	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	event.Metadata = metadataJSON

	payload, err := json.Marshal(events.NewRequestEvent(
		events.EventTypeRequestExpired,
		event.ID,
		request.ID,
		request.PassengerID,
		request.TripID,
		nil,
		event.CreatedAt,
		meta,
	))
	if err != nil {
		return err
	}
	event.Payload = payload

	if err := e.outbox.SaveTx(ctx, tx, &event); err != nil {
		return fmt.Errorf("failed to save request_expired outbox: %w", err)
	}
	return nil
}
