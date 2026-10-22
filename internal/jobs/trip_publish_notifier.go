package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
)

type TripPublishNotifier struct {
	trips    *repository.TripRepository
	requests *requestRepo.RequestRepository
	outbox   *outbox.EventRepository
}

func NewTripPublishNotifier(
	trips *repository.TripRepository,
	requests *requestRepo.RequestRepository,
	eventsRepo *outbox.EventRepository,
) *TripPublishNotifier {
	return &TripPublishNotifier{
		trips:    trips,
		requests: requests,
		outbox:   eventsRepo,
	}
}

func (n *TripPublishNotifier) Tick(ctx context.Context) error {
	logger := logctx.Logger(ctx).With(
		slog.String("job", "TripPublishNotifier"),
	)
	err := storage.TxWithoutResult(ctx, n.trips.GetDB(), func(tx pgx.Tx) error {
		trips, err := n.trips.ListPublishedForUpdateTx(ctx, tx)
		if err != nil {
			return err
		}

		for i := range trips {
			if err := n.notifyTrip(ctx, tx, trips[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("trip publish notifier tick failed", slog.Any("error", err))
	}
	return err
}

func (n *TripPublishNotifier) notifyTrip(ctx context.Context, tx pgx.Tx, trip entity.Trip) error {
	occupied, err := n.requests.SumAgreedSeatsByTripIDTx(ctx, tx, trip.ID, uuid.Nil)
	if err != nil {
		return err
	}
	if occupied >= trip.Seats {
		return nil
	}

	passengerIDs, err := n.requests.ListMatchingPublishedPassengersTx(
		ctx,
		tx,
		trip.ID,
		trip.FromPoint,
		trip.ToPoint,
		trip.DepartureTime.UTC().Format("2006-01-02"),
	)
	if err != nil {
		return err
	}
	if len(passengerIDs) == 0 {
		return nil
	}

	if err := n.requests.MarkPublishNotifiedTx(ctx, tx, trip.ID, passengerIDs); err != nil {
		return err
	}

	ids := make([]string, 0, len(passengerIDs))
	for _, id := range passengerIDs {
		ids = append(ids, id.String())
	}

	return saveTripOutbox(
		ctx,
		tx,
		n.outbox,
		trip.ID,
		outbox.EventTypeTripPublishNotificated,
		events.EventTypeTripPublishNotificated,
		func(eventID uuid.UUID, occurredAt time.Time, meta events.EventMetadata) (any, error) {
			return events.NewTripPublishNotificated(
				eventID,
				trip.ID,
				trip.DriverID,
				trip.FromPoint,
				trip.ToPoint,
				trip.DepartureTime,
				ids,
				occurredAt,
				meta,
			), nil
		},
	)
}

func saveTripOutbox(
	ctx context.Context,
	tx pgx.Tx,
	repo *outbox.EventRepository,
	aggregateID uuid.UUID,
	outboxType outbox.EventType,
	kafkaType events.EventType,
	buildPayload func(eventID uuid.UUID, occurredAt time.Time, meta events.EventMetadata) (any, error),
) error {
	event := outbox.NewPendingEvent(outboxType, aggregateID, nil)
	meta := events.MetadataFromContext(ctx, event.ID.String(), kafkaType, event.CreatedAt)

	metadataJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	event.Metadata = metadataJSON

	payloadObj, err := buildPayload(event.ID, event.CreatedAt, meta)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(payloadObj)
	if err != nil {
		return err
	}
	event.Payload = payload

	if err := repo.SaveTx(ctx, tx, &event); err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}
	return nil
}
