package api_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	requestEntity "job4j_go_share_trip/internal/business/request/entity"
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/events"
	"job4j_go_share_trip/internal/jobs"
	"job4j_go_share_trip/internal/shared/outbox"
)

func newWorkers() (*jobs.TripPublishNotifier, *jobs.RequestExpiry) {
	m := getTestMetrics()
	trips := repository.NewPostgresRepository(testPool, m)
	requests := requestRepo.NewPostgresRepository(testPool, m)
	eventsRepo := outbox.NewEventRepository(testPool, m)
	return jobs.NewTripPublishNotifier(trips, requests, eventsRepo), jobs.NewRequestExpiry(requests, eventsRepo)
}

func TestTripPublishNotifier_MatchesSameDateIgnoresTime(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	sameDayDifferentTime := time.Date(
		trip.Trip.DepartureTime.UTC().Year(),
		trip.Trip.DepartureTime.UTC().Month(),
		trip.Trip.DepartureTime.UTC().Day(),
		3, 0, 0, 0, time.UTC,
	)
	req, err := InsertTestRequestDetails(
		ctx, testPool, passengerID, requestEntity.StatusPublished, nil,
		trip.Trip.FromPoint, trip.Trip.ToPoint, sameDayDifferentTime, 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req.ID) })

	notifier, _ := newWorkers()
	require.NoError(t, notifier.Tick(ctx))

	raw, err := outboxPayloadByType(ctx, testPool, trip.TripID, "trip_publish_notificated")
	require.NoError(t, err)
	var event events.TripPublishNotificated
	require.NoError(t, json.Unmarshal(raw, &event))
	assert.Equal(t, []string{passengerID.String()}, event.PassengerIDs)
	assert.Equal(t, trip.Trip.FromPoint, event.FromPoint)
	assert.Equal(t, trip.Trip.ToPoint, event.ToPoint)

	require.NoError(t, notifier.Tick(ctx))
	var count int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'trip_publish_notificated'`,
		trip.TripID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestTripPublishNotifier_SkipsDifferentDate(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	req, err := InsertTestRequestDetails(
		ctx, testPool, passengerID, requestEntity.StatusPublished, nil,
		trip.Trip.FromPoint, trip.Trip.ToPoint, trip.Trip.DepartureTime.Add(48*time.Hour), 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req.ID) })

	notifier, _ := newWorkers()
	require.NoError(t, notifier.Tick(ctx))

	var count int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'trip_publish_notificated'`,
		trip.TripID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTripPublishNotifier_SkipsWhenNoSeats(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	_, err = testPool.Exec(ctx, `UPDATE trips SET seats = 1 WHERE id = $1`, trip.TripID)
	require.NoError(t, err)
	trip.Trip.Seats = 1
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	holder := uuid.New()
	agreed, err := InsertTestRequestDetails(
		ctx, testPool, holder, requestEntity.StatusAgreed, &trip.TripID,
		trip.Trip.FromPoint, trip.Trip.ToPoint, trip.Trip.DepartureTime, 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, agreed.ID) })

	passengerID := uuid.New()
	req, err := InsertTestRequestDetails(
		ctx, testPool, passengerID, requestEntity.StatusPublished, nil,
		trip.Trip.FromPoint, trip.Trip.ToPoint, trip.Trip.DepartureTime, 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req.ID) })

	notifier, _ := newWorkers()
	require.NoError(t, notifier.Tick(ctx))

	var count int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'trip_publish_notificated'`,
		trip.TripID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTripPublishNotifier_NotifiesNewPassengerOnly(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	first := uuid.New()
	req1, err := InsertTestRequestDetails(
		ctx, testPool, first, requestEntity.StatusPublished, nil,
		trip.Trip.FromPoint, trip.Trip.ToPoint, trip.Trip.DepartureTime, 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req1.ID) })

	notifier, _ := newWorkers()
	require.NoError(t, notifier.Tick(ctx))

	second := uuid.New()
	req2, err := InsertTestRequestDetails(
		ctx, testPool, second, requestEntity.StatusPublished, nil,
		trip.Trip.FromPoint, trip.Trip.ToPoint, trip.Trip.DepartureTime, 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req2.ID) })

	require.NoError(t, notifier.Tick(ctx))

	raw, err := outboxPayloadByType(ctx, testPool, trip.TripID, "trip_publish_notificated")
	require.NoError(t, err)
	var event events.TripPublishNotificated
	require.NoError(t, json.Unmarshal(raw, &event))
	assert.Equal(t, []string{second.String()}, event.PassengerIDs)
}

func TestRequestExpiry_CancelsYesterdayDraft(t *testing.T) {
	ctx := context.Background()
	passengerID := uuid.New()
	yesterday := time.Now().UTC().Add(-24 * time.Hour)
	req, err := InsertTestRequestDetails(
		ctx, testPool, passengerID, requestEntity.StatusDraft, nil,
		"Moscow", "Kazan", yesterday, 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req.ID) })

	_, expiry := newWorkers()
	require.NoError(t, expiry.Tick(ctx))

	repo := requestRepo.NewPostgresRepository(testPool, getTestMetrics())
	updated, err := repo.GetByID(ctx, req.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusCanceled, updated.Status)
	assert.Nil(t, updated.TripID)

	eventType, _, err := latestOutboxEvent(ctx, testPool, req.ID)
	require.NoError(t, err)
	assert.Equal(t, "request_expired", eventType)

	require.NoError(t, expiry.Tick(ctx))
	var count int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1 AND event_type = 'request_expired'`,
		req.ID,
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestRequestExpiry_SkipsTodayAndStarted(t *testing.T) {
	ctx := context.Background()
	todayPassenger := uuid.New()
	today, err := InsertTestRequestDetails(
		ctx, testPool, todayPassenger, requestEntity.StatusPublished, nil,
		"Moscow", "Kazan", time.Now().UTC().Add(2*time.Hour), 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, today.ID) })

	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusStarted)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	startedPassenger := uuid.New()
	started, err := InsertTestRequestDetails(
		ctx, testPool, startedPassenger, requestEntity.StatusStarted, &trip.TripID,
		"Moscow", "Kazan", time.Now().UTC().Add(-48*time.Hour), 1,
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, started.ID) })

	_, expiry := newWorkers()
	require.NoError(t, expiry.Tick(ctx))

	repo := requestRepo.NewPostgresRepository(testPool, getTestMetrics())
	stillToday, err := repo.GetByID(ctx, today.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusPublished, stillToday.Status)

	stillStarted, err := repo.GetByID(ctx, started.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusStarted, stillStarted.Status)
}
