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

func TestMoveTripFromPublishToStarted_Success(t *testing.T) {
	t.Run("success - перевод из Published в Started", func(t *testing.T) {
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

		payload := api.MoveTripFromPublishToStartedRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_started",
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

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusStarted, updatedTrip.Status)
	})
}

func TestMoveTripFromPublishToStarted_InvalidStatus(t *testing.T) {
	t.Run("error - поездка в невалидном статусе (409 Conflict)", func(t *testing.T) {
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

		payload := api.MoveTripFromPublishToStartedRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_started",
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

		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusDraft, updatedTrip.Status)
	})
}

func TestMoveTripFromPublishToStarted_AlreadyStarted(t *testing.T) {
	t.Run("success - поездка уже начата (204 No Content)", func(t *testing.T) {
		ctx := context.Background()

		driverID := uuid.New()
		testData, err := CreateTestTripWithStatus(
			ctx,
			testPool,
			driverID,
			entity.StatusStarted,
		)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		payload := api.MoveTripFromPublishToStartedRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_started",
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

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestMoveTripFromPublishToStarted_DriverNotMatch(t *testing.T) {
	t.Run("forbidden - driver_id не совпадает", func(t *testing.T) {
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

		otherClientID := uuid.New()
		payload := api.MoveTripFromPublishToStartedRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_started",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		token := testutils.GenerateTestToken(
			otherClientID.String(),
			"otheruser",
			"other@example.com",
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
	})
}

func TestMoveTripFromPublishToStarted_TripNotFound(t *testing.T) {
	t.Run("error - поездка не найдена", func(t *testing.T) {
		driverID := uuid.New()
		payload := api.MoveTripFromPublishToStartedRequest{
			TripID: uuid.New(),
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_started",
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

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}