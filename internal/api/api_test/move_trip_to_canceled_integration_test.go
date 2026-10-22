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

func TestMoveTripToCanceled_FromAnyStatus(t *testing.T) {
	statuses := []entity.Status{
		entity.StatusDraft,
		entity.StatusPreapproved,
		entity.StatusPublished,
		entity.StatusConfirmed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			ctx := context.Background()
			driverID := uuid.New()
			testData, err := CreateTestTripWithStatus(ctx, testPool, driverID, status)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = CleanupTestData(ctx, testPool, testData)
			})

			token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
			resp := requestJSON(t, http.MethodPut, "/trip/move_to_canceled", token, api.MoveTripToCanceledRequest{
				TripID:    testData.TripID,
				CompanyID: testCompanyID,
			})
			defer closeBody(t, resp)

			require.Equal(t, http.StatusOK, resp.StatusCode)

			var response api.MoveTripToCanceledResponseWrapper
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

			expected := api.MoveTripToCanceledResponseWrapper{
				Status: "Success",
				Data: api.MoveTripToCanceledResponse{
					ID:            testData.TripID.String(),
					DriverID:      driverID.String(),
					FromPoint:     testData.Trip.FromPoint,
					ToPoint:       testData.Trip.ToPoint,
					Seats:         testData.Trip.Seats,
					Status:        string(entity.StatusCanceled),
					CreatedAt:     response.Data.CreatedAt,
					DepartureTime: response.Data.DepartureTime,
				},
			}
			assert.Equal(t, expected, response)

			tripRepo := repository.NewPostgresRepository(testPool, getTestMetrics())
			updated, err := tripRepo.GetByTripID(ctx, testData.TripID)
			require.NoError(t, err)
			assert.Equal(t, entity.StatusCanceled, updated.Status)

			eventType, eventStatus, err := latestOutboxEvent(ctx, testPool, testData.TripID)
			require.NoError(t, err)
			assert.Equal(t, "trip_canceled", eventType)
			assert.Equal(t, "pending", eventStatus)
		})
	}
}

func TestMoveTripToCanceled_StartedForbidden(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	testData, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusStarted)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_canceled", token, api.MoveTripToCanceledRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var response api.ErrorResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	require.Equal(t, "Error", response.Status)
	require.Contains(t, response.Message, "Started trip cannot be canceled")

	tripRepo := repository.NewPostgresRepository(testPool, getTestMetrics())
	updated, err := tripRepo.GetByTripID(ctx, testData.TripID)
	require.NoError(t, err)
	assert.Equal(t, entity.StatusStarted, updated.Status)
}

func TestMoveTripToCanceled_AlreadyCanceled(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	testData, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusCanceled)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_canceled", token, api.MoveTripToCanceledRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMoveTripToCanceled_DriverNotMatch(t *testing.T) {
	ctx := context.Background()
	testData, err := CreateTestTripWithStatus(ctx, testPool, uuid.New(), entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, testData)
	})

	token := testutils.GenerateTestToken(uuid.New().String(), "other", "o@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_canceled", token, api.MoveTripToCanceledRequest{
		TripID:    testData.TripID,
		CompanyID: testCompanyID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	var response api.ErrorResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	require.Equal(t, "Error", response.Status)
	require.Contains(t, response.Message, "is not driver")
}

func TestMoveTripToCanceled_TripNotFound(t *testing.T) {
	token := testutils.GenerateTestToken(uuid.New().String(), "driver", "d@example.com")
	resp := requestJSON(t, http.MethodPut, "/trip/move_to_canceled", token, api.MoveTripToCanceledRequest{
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
