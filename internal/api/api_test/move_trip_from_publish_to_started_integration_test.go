// internal/api/api_test/move_trip_from_publish_to_started_integration_test.go
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
	"go.uber.org/mock/gomock"

	"job4j_go_share_trip/internal/api"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/business/trip/service/mocks"
	"job4j_go_share_trip/internal/clients/contract"
	"job4j_go_share_trip/internal/shared/outbox"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func newTestTripService(contractClient service.ContractClient) *service.TripService {
	m := getTestMetrics()
	tripRepo := repository.NewPostgresRepository(testPool, m)
	eventRepo := outbox.NewEventRepository(testPool, m)
	tripDomain := domain.NewDomain(*tripRepo, *eventRepo, m)

	return service.NewService(
		*tripDomain,
		*tripRepo,
		*eventRepo,
		contractClient,
		nil,
		m,
	)
}

func moveFromPublishToStartedRequest(data *TestData) service.MoveFromPublishToStartedRequest {
	return service.MoveFromPublishToStartedRequest{
		TripID:    data.TripID,
		ClientID:  data.DriverID,
		CompanyID: testCompanyID,
	}
}

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
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
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

		var response api.MoveTripFromPublishToStartedResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		expected := api.MoveTripFromPublishToStartedResponseWrapper{
			Status: "Success",
			Data: api.MoveTripFromPublishToStartedResponse{
				ID:            testData.TripID.String(),
				DriverID:      driverID.String(),
				FromPoint:     testData.Trip.FromPoint,
				ToPoint:       testData.Trip.ToPoint,
				Seats:         testData.Trip.Seats,
				Status:        string(entity.StatusStarted),
				CreatedAt:     response.Data.CreatedAt,
				DepartureTime: response.Data.DepartureTime,
			},
		}
		assert.Equal(t, expected, response)

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
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
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

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Error", response.Status)
		require.Contains(t, response.Message, "invalid trip status")

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
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
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
		// 204 No Content — тела нет, декодировать нечего
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
			TripID:    testData.TripID,
			CompanyID: testCompanyID,
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

		var response api.ErrorResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		require.Equal(t, "Error", response.Status)
		require.Contains(t, response.Message, "is not driver")
	})
}

func TestMoveTripFromPublishToStarted_TripNotFound(t *testing.T) {
	t.Run("error - поездка не найдена", func(t *testing.T) {
		driverID := uuid.New()
		payload := api.MoveTripFromPublishToStartedRequest{
			TripID:    uuid.New(),
			CompanyID: testCompanyID,
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

func TestService_StartTrip_Allowed(t *testing.T) {
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

	ctrl := gomock.NewController(t)
	contractClient := mocks.NewMockContractClient(ctrl)

	contractClient.EXPECT().
		CheckService(gomock.Any(), testCompanyID.String(), entity.ServiceTripStart).
		Return(contract.CheckResult{Allowed: true, Reason: "service_allowed"}, nil)

	svc := newTestTripService(contractClient)

	resp, err := svc.MoveFromPublishToStarted(ctx, moveFromPublishToStartedRequest(testData))

	require.NoError(t, err)
	require.NotNil(t, resp)

	expected := service.MoveFromPublishToStartedResponse{
		ID:             testData.TripID,
		DriverID:       driverID,
		FromPoint:      testData.Trip.FromPoint,
		ToPoint:        testData.Trip.ToPoint,
		Seats:          testData.Trip.Seats,
		Status:         string(entity.StatusStarted),
		CreatedAt:      resp.CreatedAt,
		DepartureTime:  resp.DepartureTime,
		AlreadyStarted: false,
	}
	assert.Equal(t, expected, *resp)
}

func TestService_StartTrip_Denied(t *testing.T) {
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

	ctrl := gomock.NewController(t)
	contractClient := mocks.NewMockContractClient(ctrl)

	contractClient.EXPECT().
		CheckService(gomock.Any(), testCompanyID.String(), entity.ServiceTripStart).
		Return(contract.CheckResult{Allowed: false, Reason: "service_not_allowed"}, nil)

	svc := newTestTripService(contractClient)

	_, err = svc.MoveFromPublishToStarted(ctx, moveFromPublishToStartedRequest(testData))

	require.ErrorIs(t, err, tripErrors.ErrTripStartIsNotAllowed)

	m := getTestMetrics()
	tripRepo := repository.NewPostgresRepository(testPool, m)
	updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
	require.NoError(t, err)
	assert.Equal(t, entity.StatusPublished, updatedTrip.Status)
}

func TestService_StartTrip_Timeout(t *testing.T) {
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

	ctrl := gomock.NewController(t)
	contractClient := mocks.NewMockContractClient(ctrl)

	contractClient.EXPECT().
		CheckService(gomock.Any(), testCompanyID.String(), entity.ServiceTripStart).
		Return(contract.CheckResult{}, context.DeadlineExceeded)

	svc := newTestTripService(contractClient)

	_, err = svc.MoveFromPublishToStarted(ctx, moveFromPublishToStartedRequest(testData))

	require.ErrorIs(t, err, tripErrors.ErrTripStartIsNotAllowed)
	require.Contains(t, err.Error(), context.DeadlineExceeded.Error())

	m := getTestMetrics()
	tripRepo := repository.NewPostgresRepository(testPool, m)
	updatedTrip, err := tripRepo.GetByTripID(ctx, testData.TripID)
	require.NoError(t, err)
	assert.Equal(t, entity.StatusPublished, updatedTrip.Status)
}
