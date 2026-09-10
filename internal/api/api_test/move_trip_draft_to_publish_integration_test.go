// internal/api/api_test/move_trip_draft_to_publish_integration_test.go
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

func TestMoveTripFromDraftToPublished_Success(t *testing.T) {
	t.Run("success - перевод из Draft в Published", func(t *testing.T) {
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

		payload := api.MoveTripDraftToPublishModelRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_publish",
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

		var response api.MoveTripDraftToPublishResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Success", response.Status)
		require.Equal(t, testData.TripID.String(), response.Data.TripID)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusPublished, updatedTrip.Status)
	})
}

func TestMoveTripFromDraftToPublished_DriverNotMatch(t *testing.T) {
	t.Run("forbidden - driver_id не совпадает", func(t *testing.T) {
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

		otherClientID := uuid.New()
		payload := api.MoveTripDraftToPublishModelRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_publish",
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

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Error", response.Status)
		require.NotEmpty(t, response.Message)

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusDraft, updatedTrip.Status)
	})
}

func TestMoveTripFromDraftToPublished_TripNotFound(t *testing.T) {
	t.Run("error - поездка не найдена", func(t *testing.T) {
		driverID := uuid.New()
		payload := api.MoveTripDraftToPublishModelRequest{
			TripID: uuid.New(),
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_publish",
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

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Error", response.Status)
		require.Equal(t, "trip not found", response.Message)
	})
}

func TestMoveTripFromDraftToPublished_AlreadyPublished(t *testing.T) {
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

		payload := api.MoveTripDraftToPublishModelRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_publish",
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
		// 204 No Content — тела нет, декодировать нечего

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusPublished, updatedTrip.Status)
	})
}

func TestMoveTripFromDraftToPublished_InvalidStatus(t *testing.T) {
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

		invalidStatus := entity.Status("canceled")

		_, err = testPool.Exec(ctx,
			`UPDATE trips SET status = $1 WHERE id = $2`,
			invalidStatus,
			testData.TripID,
		)
		require.NoError(t, err)

		payload := api.MoveTripDraftToPublishModelRequest{
			TripID: testData.TripID,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPut,
			"/trip/move_to_publish",
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

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Error", response.Status)
		require.Contains(t, response.Message, "invalid trip status")

		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, invalidStatus, updatedTrip.Status)
	})
}