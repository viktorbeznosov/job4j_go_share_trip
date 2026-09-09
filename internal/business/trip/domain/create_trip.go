package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/storage"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateTripRequest struct {
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

type CreateTripResponse struct {
	ID            uuid.UUID
	DriverID      uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        string
	CreatedAt     time.Time
}

func (d *TripDomain) Create(ctx context.Context, req CreateTripRequest) (*CreateTripResponse, error) {
	started := time.Now()
	result := "success"

	logger := logctx.Logger(ctx).With(
		slog.String("domain", "TripDomain"),
		slog.String("operation", "CreateTrip"),
		slog.String("client_id", req.DriverID.String()),
	)

	trip, err := entity.NewTrip(
		req.DriverID,
		req.FromPoint,
		req.ToPoint,
		req.DepartureTime,
		req.Seats,
	)
	if err != nil {
		logger.Error("failed to create trip entity", slog.Any("error", err))
		return nil, err
	}

	err = storage.TxWithoutResult(ctx, d.tripRepository.GetDB(), func(tx pgx.Tx) error {
		if err := d.tripRepository.CreateTx(ctx, tx, trip); err != nil {
			logger.Error("failed to create trip in transaction", slog.Any("error", err))
			return err
		}

		if err := d.tripRepository.CreateHistoryTx(ctx, tx, trip.ID, nil, &trip.Status); err != nil {
			logger.Error("failed to create history in transaction", slog.Any("error", err))
			return err
		}

		payload, err := json.Marshal(trip)
		if err != nil {
			return err
		}

		event := outbox.Event{
			ID:          uuid.New(),
			EventName:   outbox.TripCreated,
			AggregateID: trip.ID,
			Payload:     payload,
			CreatedAt:   time.Now(),
		}

		if err := d.eventRepository.SaveTx(ctx, tx, &event); err != nil {
			logger.Error("failed to save outbox event", slog.Any("error", err))
			result = "error"
			d.metrics.TripCreateTotal.WithLabelValues(result).Inc()
			d.metrics.TripCreateDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
			return fmt.Errorf("failed to save outbox event: %w", err)
		}

		return nil
	})

	if err != nil {
		logger.Error("failed to create trip", slog.Any("error", err))
		return nil, err
	}

	logger.Info("create trip completed", slog.String("trip_id", trip.ID.String()))

	return &CreateTripResponse{
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