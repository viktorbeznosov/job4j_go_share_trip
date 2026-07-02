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
)

func TestMoveTripFromDraftToPublished_Success(t *testing.T) {
	t.Run("success - перевод из Draft в Published", func(t *testing.T) {
		ctx := context.Background()

		// 1. Создаём тестовую поездку в БД
		driverID := uuid.New()
		testData, err := CreateTestTrip(ctx, testPool, driverID)
		require.NoError(t, err)

		// ✅ Исправлено: проверяем ошибку при очистке
		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		// 2. Формируем запрос
		payload := request.MoveTripDraftToPublishModelRequest{
			TripID:   testData.TripID,
			ClientID: driverID,
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

		// 3. Выполняем запрос
		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		// ✅ Исправлено: проверяем ошибку при закрытии
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		// 4. Проверяем ответ
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 5. Проверяем, что статус изменился в БД
		tripRepo := repository.NewPostgresRepository(testPool)
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

		// ✅ Исправлено
		defer func() {
			err := CleanupTestData(ctx, testPool, testData)
			if err != nil {
				t.Errorf("failed to cleanup test data: %v", err)
			}
		}()

		// 2. Запрос от другого пользователя
		otherClientID := uuid.New()
		payload := request.MoveTripDraftToPublishModelRequest{
			TripID:   testData.TripID,
			ClientID: otherClientID,
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

		resp, err := testApp.Test(req, -1)
		require.NoError(t, err)

		// ✅ Исправлено
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		// 3. Должна быть ошибка 403 Forbidden
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// 4. Статус не должен измениться
		tripRepo := repository.NewPostgresRepository(testPool)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusDraft, updatedTrip.Status)
	})
}

func TestMoveTripFromDraftToPublished_TripNotFound(t *testing.T) {
	t.Run("error - поездка не найдена", func(t *testing.T) {
		// 1. Запрос с несуществующим ID
		payload := request.MoveTripDraftToPublishModelRequest{
			TripID:   uuid.New(),
			ClientID: uuid.New(),
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

// TestMoveTripFromDraftToPublished_AlreadyPublished
// Проверяет случай, когда пытаются опубликовать уже опубликованную поездку
// Ожидаемый результат: статус 204 No Content
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
			TripID:   testData.TripID,
			ClientID: driverID,
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
		tripRepo := repository.NewPostgresRepository(testPool)
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, entity.StatusPublished, updatedTrip.Status)
	})
}

// TestMoveTripFromDraftToPublished_InvalidStatus
// Проверяет случай, когда поездка находится в невалидном статусе (не Draft и не Published)
// Ожидаемый результат: статус 409 Conflict
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

		// 2. Меняем статус в БД на "CANCELLED" (или другой невалидный статус)
		// Предполагаем, что есть такой статус, если нет - используем Published
		// Но мы хотим протестировать случай, когда статус не Draft и не Published
		// Поэтому сначала опубликуем, а потом "откатим" вручную через SQL
		invalidStatus := entity.Status("canceled")

		tripRepo := repository.NewPostgresRepository(testPool)

		// Обновляем статус напрямую через SQL
		_, err = testPool.Exec(ctx,
			`UPDATE trips SET status = $1 WHERE id = $2`,
			invalidStatus,
			testData.TripID,
		)
		require.NoError(t, err)

		// 3. Пытаемся опубликовать поездку в невалидном статусе
		payload := request.MoveTripDraftToPublishModelRequest{
			TripID:   testData.TripID,
			ClientID: driverID,
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
		updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
		require.NoError(t, err)
		assert.Equal(t, invalidStatus, updatedTrip.Status)
	})
}