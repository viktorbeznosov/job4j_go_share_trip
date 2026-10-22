package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"job4j_go_share_trip/internal/api"
	requestEntity "job4j_go_share_trip/internal/business/request/entity"
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/events"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func TestMoveTripToCanceled_DetachesAttachedRequests(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	p1 := uuid.New()
	p2 := uuid.New()
	req1, err := InsertTestRequest(ctx, testPool, p1, requestEntity.StatusAgreed, &trip.TripID)
	require.NoError(t, err)
	req2, err := InsertTestRequest(ctx, testPool, p2, requestEntity.StatusPreAgreed, &trip.TripID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupRequest(ctx, testPool, req1.ID)
		_ = CleanupRequest(ctx, testPool, req2.ID)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_canceled", token, api.MoveTripToCanceledRequest{
		TripID:    trip.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	repo := requestRepo.NewPostgresRepository(testPool, getTestMetrics())
	updated1, err := repo.GetByID(ctx, req1.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusPublished, updated1.Status)
	assert.Nil(t, updated1.TripID)

	updated2, err := repo.GetByID(ctx, req2.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusPublished, updated2.Status)
	assert.Nil(t, updated2.TripID)

	raw, err := outboxPayloadByType(ctx, testPool, trip.TripID, "trip_canceled")
	require.NoError(t, err)
	var canceled events.TripCanceled
	require.NoError(t, json.Unmarshal(raw, &canceled))
	assert.ElementsMatch(t, []string{p1.String(), p2.String()}, canceled.PassengerIDs)
	assert.ElementsMatch(t, []string{req1.ID.String(), req2.ID.String()}, canceled.RequestIDs)
}

func TestMoveTripToStarted_MovesAgreedRequestsAndDetachesPreAgreed(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	agreedPassenger := uuid.New()
	prePassenger := uuid.New()
	agreed, err := InsertTestRequest(ctx, testPool, agreedPassenger, requestEntity.StatusAgreed, &trip.TripID)
	require.NoError(t, err)
	pre, err := InsertTestRequest(ctx, testPool, prePassenger, requestEntity.StatusPreAgreed, &trip.TripID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupRequest(ctx, testPool, agreed.ID)
		_ = CleanupRequest(ctx, testPool, pre.ID)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_started", token, api.MoveTripFromPublishToStartedRequest{
		TripID:    trip.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	repo := requestRepo.NewPostgresRepository(testPool, getTestMetrics())
	updatedAgreed, err := repo.GetByID(ctx, agreed.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusStarted, updatedAgreed.Status)
	require.NotNil(t, updatedAgreed.TripID)
	assert.Equal(t, trip.TripID, *updatedAgreed.TripID)

	updatedPre, err := repo.GetByID(ctx, pre.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusPublished, updatedPre.Status)
	assert.Nil(t, updatedPre.TripID)

	eventType, _, err := latestOutboxEvent(ctx, testPool, agreed.ID)
	require.NoError(t, err)
	assert.Equal(t, "request_started", eventType)
}

func TestMoveTripToCompleted_CompletesStartedRequests(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusStarted)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	req, err := InsertTestRequest(ctx, testPool, passengerID, requestEntity.StatusStarted, &trip.TripID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, req.ID) })

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_completed", token, api.MoveTripToCompletedRequest{
		TripID:    trip.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	repo := requestRepo.NewPostgresRepository(testPool, getTestMetrics())
	updated, err := repo.GetByID(ctx, req.ID)
	require.NoError(t, err)
	assert.Equal(t, requestEntity.StatusCompleted, updated.Status)
	require.NotNil(t, updated.TripID)

	eventType, _, err := latestOutboxEvent(ctx, testPool, req.ID)
	require.NoError(t, err)
	assert.Equal(t, "request_completed", eventType)
}
