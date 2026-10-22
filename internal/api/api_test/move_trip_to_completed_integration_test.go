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
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func TestMoveTripToCompleted_Success(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	testData, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusStarted)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_completed", token, api.MoveTripToCompletedRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response api.MoveTripToCompletedResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	expected := api.MoveTripToCompletedResponseWrapper{
		Status: "Success",
		Data: api.MoveTripToCompletedResponse{
			ID:            testData.TripID.String(),
			DriverID:      driverID.String(),
			FromPoint:     testData.Trip.FromPoint,
			ToPoint:       testData.Trip.ToPoint,
			Seats:         testData.Trip.Seats,
			Status:        string(entity.StatusConfirmed),
			CreatedAt:     response.Data.CreatedAt,
			DepartureTime: response.Data.DepartureTime,
		},
	}
	assert.Equal(t, expected, response)

	tripRepo := repository.NewPostgresRepository(testPool, getTestMetrics())
	updated, err := tripRepo.GetByTripID(ctx, testData.TripID)
	require.NoError(t, err)
	assert.Equal(t, entity.StatusConfirmed, updated.Status)

	eventType, eventStatus, err := latestOutboxEvent(ctx, testPool, testData.TripID)
	require.NoError(t, err)
	assert.Equal(t, "trip_completed", eventType)
	assert.Equal(t, "pending", eventStatus)
}

func TestMoveTripToCompleted_InvalidStatus(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	testData, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_completed", token, api.MoveTripToCompletedRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var response api.ErrorResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	require.Equal(t, "Error", response.Status)
	require.Contains(t, response.Message, "invalid trip status")
}

func TestMoveTripToCompleted_AlreadyCompleted(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	testData, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusConfirmed)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_completed", token, api.MoveTripToCompletedRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMoveTripToCompleted_DriverNotMatch(t *testing.T) {
	ctx := context.Background()
	testData, err := CreateTestTripWithStatus(ctx, testPool, uuid.New(), entity.StatusStarted)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(uuid.New().String(), "other", "o@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_completed", token, api.MoveTripToCompletedRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestMoveTripToCompleted_TripNotFound(t *testing.T) {
	token := testutils.GenerateTestToken(uuid.New().String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_completed", token, api.MoveTripToCompletedRequest{
		TripID:    uuid.New(),
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var response api.ErrorResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	assert.Equal(t, api.ErrorResponseWrapper{
		Status:  "Error",
		Message: "trip not found",
	}, response)
}
