package api_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	requestEntity "job4j_go_share_trip/internal/business/request/entity"
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
	contractclient "job4j_go_share_trip/internal/clients/contract"
	"job4j_go_share_trip/internal/observability/metrics"
)

type TestData struct {
	TripID   uuid.UUID
	DriverID uuid.UUID
	Trip     *entity.Trip
}

var testCompanyID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

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
	_, err := pool.Exec(ctx, `DELETE FROM trip_publish_notified WHERE trip_id = $1`, data.TripID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `DELETE FROM trip_requests WHERE trip_id = $1`, data.TripID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `DELETE FROM outbox_events WHERE aggregate_id = $1`, data.TripID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `DELETE FROM trips WHERE id = $1`, data.TripID)
	return err
}

func InsertTestRequest(
	ctx context.Context,
	pool *pgxpool.Pool,
	passengerID uuid.UUID,
	status requestEntity.Status,
	tripID *uuid.UUID,
) (*requestEntity.Request, error) {
	return InsertTestRequestDetails(ctx, pool, passengerID, status, tripID, "Moscow", "Kazan", time.Now().Add(24*time.Hour), 1)
}

func InsertTestRequestDetails(
	ctx context.Context,
	pool *pgxpool.Pool,
	passengerID uuid.UUID,
	status requestEntity.Status,
	tripID *uuid.UUID,
	fromPoint, toPoint string,
	departureTime time.Time,
	seats int,
) (*requestEntity.Request, error) {
	m := getTestMetrics()
	repo := requestRepo.NewPostgresRepository(pool, m)

	item := &requestEntity.Request{
		ID:            uuid.New(),
		PassengerID:   passengerID,
		FromPoint:     fromPoint,
		ToPoint:       toPoint,
		DepartureTime: departureTime,
		Seats:         seats,
		Status:        requestEntity.StatusDraft,
		CreatedAt:     time.Now(),
	}

	if err := repo.Create(ctx, item); err != nil {
		return nil, err
	}

	if status != requestEntity.StatusDraft || tripID != nil {
		item.Status = status
		item.TripID = tripID
		if err := repo.Update(ctx, item); err != nil {
			return nil, err
		}
	}

	return item, nil
}

func CleanupRequest(ctx context.Context, pool *pgxpool.Pool, requestID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM outbox_events WHERE aggregate_id = $1`, requestID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `DELETE FROM trip_requests WHERE id = $1`, requestID)
	return err
}

func latestOutboxEvent(ctx context.Context, pool *pgxpool.Pool, tripID uuid.UUID) (eventType string, status string, err error) {
	err = pool.QueryRow(
		ctx,
		`SELECT event_type, status FROM outbox_events WHERE aggregate_id = $1 ORDER BY created_at DESC LIMIT 1`,
		tripID,
	).Scan(&eventType, &status)
	return eventType, status, err
}

func outboxPayloadByType(
	ctx context.Context,
	pool *pgxpool.Pool,
	aggregateID uuid.UUID,
	eventType string,
) ([]byte, error) {
	var payload []byte
	err := pool.QueryRow(
		ctx,
		`SELECT payload FROM outbox_events
		 WHERE aggregate_id = $1 AND event_type = $2
		 ORDER BY created_at DESC LIMIT 1`,
		aggregateID,
		eventType,
	).Scan(&payload)
	return payload, err
}

type MockContractClient struct {
	Allowed bool
	Reason  string
	Err     error
}

func NewMockContractClient(allowed bool) *MockContractClient {
	return &MockContractClient{Allowed: allowed}
}

func (m *MockContractClient) CheckService(
	_ context.Context,
	_ string,
	_ entity.ServiceType,
) (contractclient.CheckResult, error) {
	if m.Err != nil {
		return contractclient.CheckResult{}, m.Err
	}
	return contractclient.CheckResult{
		Allowed: m.Allowed,
		Reason:  m.Reason,
	}, nil
}
