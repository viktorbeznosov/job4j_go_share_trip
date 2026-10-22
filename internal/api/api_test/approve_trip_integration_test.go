package api_test

import (
	"bytes"
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

func TestApproveTrip_Success(t *testing.T) {
	t.Run("success - перевод из Preapproved в Published", func(t *testing.T) {
		ctx := context.Background()

		driverID := uuid.New()
		testData, err := CreateTestTripWithStatus(
			ctx,
			testPool,
			driverID,
			entity.StatusPreapproved,
		)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		payload := api.ApproveTripRequest{
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/approve",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		directorID := uuid.New()
		token := testutils.GenerateDirectorToken(
			directorID.String(),
			"director",
			"director@example.com",
		)
		req.Header.Set("X-Refresh-Token", token)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response api.ApproveTripResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		expected := api.ApproveTripResponseWrapper{
			Status: "Success",
			Data: api.ApproveTripResponse{
				ID:            testData.TripID.String(),
				DriverID:      driverID.String(),
				FromPoint:     testData.Trip.FromPoint,
				ToPoint:       testData.Trip.ToPoint,
				Seats:         testData.Trip.Seats,
				Status:        string(entity.StatusPublished),
				CreatedAt:     response.Data.CreatedAt,
				DepartureTime: response.Data.DepartureTime,
			},
		}
		assert.Equal(t, expected, response)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusPublished, updatedTrip.Status)

		eventType, eventStatus, err := latestOutboxEvent(ctx, testPool, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, "trip_published", eventType)
		assert.Equal(t, "pending", eventStatus)
	})
}

func TestApproveTrip_DriverForbidden(t *testing.T) {
	t.Run("forbidden - водитель не может аппрувить поездку", func(t *testing.T) {
		ctx := context.Background()

		driverID := uuid.New()
		testData, err := CreateTestTripWithStatus(
			ctx,
			testPool,
			driverID,
			entity.StatusPreapproved,
		)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		payload := api.ApproveTripRequest{
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/approve",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		token := testutils.GenerateTestToken(
			driverID.String(),
			"testuser",
			"test@example.com",
		)
		req.Header.Set("X-Refresh-Token", token)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusPreapproved, updatedTrip.Status)
	})
}

func TestApproveTrip_AlreadyPublished(t *testing.T) {
	t.Run("success - поездка уже опубликована (204 No Content)", func(t *testing.T) {
		ctx := context.Background()

		driverID := uuid.New()
		testData, err := CreateTestTripWithStatus(
			ctx,
			testPool,
			driverID,
			entity.StatusPublished,
		)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		payload := api.ApproveTripRequest{
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/approve",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		token := testutils.GenerateDirectorToken(
			uuid.New().String(),
			"director",
			"director@example.com",
		)
		req.Header.Set("X-Refresh-Token", token)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusPublished, updatedTrip.Status)
	})
}

func TestApproveTrip_InvalidStatus(t *testing.T) {
	t.Run("error - поездка в draft (409 Conflict)", func(t *testing.T) {
		ctx := context.Background()

		driverID := uuid.New()
		testData, err := CreateTestTrip(ctx, testPool, driverID)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		payload := api.ApproveTripRequest{
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/approve",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		token := testutils.GenerateDirectorToken(
			uuid.New().String(),
			"director",
			"director@example.com",
		)
		req.Header.Set("X-Refresh-Token", token)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Error", response.Status)
		require.Contains(t, response.Message, "invalid trip status")
	})
}

func TestApproveTrip_TripNotFound(t *testing.T) {
	t.Run("error - поездка не найдена", func(t *testing.T) {
		payload := api.ApproveTripRequest{
			TripID:    uuid.New(),
			CompanyID: testCompanyID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/approve",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		token := testutils.GenerateDirectorToken(
			uuid.New().String(),
			"director",
			"director@example.com",
		)
		req.Header.Set("X-Refresh-Token", token)

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		expected := api.ErrorResponseWrapper{
			Status:  "Error",
			Message: "trip not found",
		}
		assert.Equal(t, expected, response)
	})
}
