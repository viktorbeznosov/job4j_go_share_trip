package api_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/observability/metrics"
)

type TestData struct {
	TripID   uuid.UUID
	DriverID uuid.UUID
	Trip     *entity.Trip
}

func getTestMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()
	return metrics.New(registry)
}

func CreateTestTrip(ctx context.Context, pool *pgxpool.Pool, driverID uuid.UUID) (*TestData, error) {
	m := getTestMetrics()
	tripRepo := repository.NewPostgresRepository(pool, m)

	trip := &entity.Trip{
		ID:            uuid.New(),
		DriverID:      driverID,
		FromPoint:     "Moscow",
		ToPoint:       "Saint Petersburg",
		DepartureTime: time.Now().Add(24 * time.Hour),
		Seats:         4,
		Status:        entity.StatusDraft,
		CreatedAt:     time.Now(),
	}

	err := tripRepo.Create(ctx, trip)
	if err != nil {
		return nil, err
	}

	return &TestData{
		TripID:   trip.ID,
		DriverID: driverID,
		Trip:     trip,
	}, nil
}

func CreateTestTripWithStatus(
	ctx context.Context,
	pool *pgxpool.Pool,
	driverID uuid.UUID,
	status entity.Status,
) (*TestData, error) {
	data, err := CreateTestTrip(ctx, pool, driverID)
	if err != nil {
		return nil, err
	}

	if status != entity.StatusDraft {
		data.Trip.Status = status
		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(pool, m)

		err := tripRepo.Update(ctx, data.Trip)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func CleanupTestData(ctx context.Context, pool *pgxpool.Pool, data *TestData) error {
	_, err := pool.Exec(ctx, `DELETE FROM trips WHERE id = $1`, data.TripID)
	return err
}