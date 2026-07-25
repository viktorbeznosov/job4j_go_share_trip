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

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/handler/request"
	"job4j_go_share_trip/internal/domain/trip/repository"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func TestMoveTripFromDraftToPublished_Success(t *testing.T) {
	t.Run("success - перевод из Draft в Published", func(t *testing.T) {
		ctx := context.Background()

		// 1. Создаём тестовую поездку в БД
		driverID := uuid.New()
		testData, err := CreateTestTrip(ctx, testPool, driverID)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		// 2. Формируем запрос (ClientID теперь в токене)
		payload := request.MoveTripDraftToPublishModelRequest{
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

		// ✅ Добавляем токен с правильным driverID
		token := testutils.GenerateTestToken(
			driverID.String(),
			"testuser",
			"test@example.com",
		)
		req.Header.Set("X-Refresh-Token", token)

		// 3. Выполняем запрос
		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		// 4. Проверяем ответ
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 5. Проверяем, что статус изменился в БД
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

		// 1. Создаём поездку с одним водителем
		driverID := uuid.New()
		testData, err := CreateTestTrip(ctx, testPool, driverID)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		// 2. Запрос с токеном другого пользователя (не driver)
		otherClientID := uuid.New()
		payload := request.MoveTripDraftToPublishModelRequest{
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

		// ✅ Токен другого пользователя
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

		// 3. Должна быть ошибка 403 Forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// 4. Статус не должен измениться
		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusDraft, updatedTrip.Status)
	})
}

func TestMoveTripFromDraftToPublished_TripNotFound(t *testing.T) {
	t.Run("error - поездка не найдена", func(t *testing.T) {
		// 1. Запрос с несуществующим ID
		driverID := uuid.New()
		payload := request.MoveTripDraftToPublishModelRequest{
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

		// ✅ Добавляем токен
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

		// 2. Должна быть ошибка 404 Not Found
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestMoveTripFromDraftToPublished_AlreadyPublished(t *testing.T) {
	t.Run("success - поездка уже опубликована (204 No Content)", func(t *testing.T) {
		ctx := context.Background()

		// 1. Создаём поездку со статусом Published
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

		// 2. Пытаемся опубликовать уже опубликованную поездку
		payload := request.MoveTripDraftToPublishModelRequest{
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

		// ✅ Токен водителя
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

		// 3. Проверяем статус 204 No Content
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		// 4. Проверяем, что статус остался Published
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

		// 1. Создаём поездку со статусом Draft
		driverID := uuid.New()
		testData, err := CreateTestTrip(ctx, testPool, driverID)
		require.NoError(t, err)

		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		// 2. Меняем статус в БД на "canceled"
		invalidStatus := entity.Status("canceled")

		// Обновляем статус напрямую через SQL
		_, err = testPool.Exec(ctx,
			`UPDATE trips SET status = $1 WHERE id = $2`,
			invalidStatus,
			testData.TripID,
		)
		require.NoError(t, err)

		// 3. Пытаемся опубликовать поездку в невалидном статусе
		payload := request.MoveTripDraftToPublishModelRequest{
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

		// ✅ Токен водителя
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

		// 4. Проверяем статус 409 Conflict
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		// 5. Проверяем, что статус не изменился
		m := getTestMetrics()
		tripRepo := repository.NewPostgresRepository(testPool, m)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, invalidStatus, updatedTrip.Status)
	})
}