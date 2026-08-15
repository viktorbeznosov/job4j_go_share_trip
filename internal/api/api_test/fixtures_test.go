package api_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/repository"
	"job4j_go_share_trip/internal/observability/metrics"
)

// TestData содержит тестовые данные для использования в тестах
type TestData struct {
	TripID   uuid.UUID
	DriverID uuid.UUID
	Trip     *entity.Trip
}

// getTestMetrics создаёт тестовые метрики (пустой реестр)
func getTestMetrics() *metrics.Metrics {
	registry := prometheus.NewRegistry()
	return metrics.New(registry)
}

// CreateTestTrip создаёт тестовую поездку в БД
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

// CreateTestTripWithStatus создаёт тестовую поездку с заданным статусом
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

	// Если нужен Published статус - обновляем
	if status == entity.StatusPublished {
		data.Trip.Status = status
		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(pool, m)

		// Обновляем статус в БД
		err := tripRepo.UpdateTx(ctx, pool, data.Trip)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

// CleanupTestData удаляет тестовые данные
func CleanupTestData(ctx context.Context, pool *pgxpool.Pool, data *TestData) error {
	_, err := pool.Exec(ctx, `DELETE FROM trips WHERE id = $1`, data.TripID)
	return err
}