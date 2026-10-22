package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"job4j_go_share_trip/internal/api"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/events"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func requestJSON(
	t *testing.T,
	method string,
	path string,
	token string,
	payload any,
) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Refresh-Token", token)

	resp, err := testApp.Test(req, -1)
	require.NoError(t, err)
	return resp
}

func createPublishedRequest(
	t *testing.T,
	passengerID uuid.UUID,
	seats int,
) api.RequestResponse {
	t.Helper()

	token := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	createResp := requestJSON(t, http.MethodPost, "/request/", token, api.CreateRequestRequest{
		FromPoint:     "Moscow",
		ToPoint:       "Kazan",
		DepartureTime: time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04"),
		Seats:         seats,
	})
	defer closeBody(t, createResp)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)

	var created api.CreateRequestResponseWrapper
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	publishResp := requestJSON(t, http.MethodPut, "/request/move_to_publish", token, api.MoveRequestToPublishRequest{
		RequestID: uuid.MustParse(created.Data.ID),
	})
	defer closeBody(t, publishResp)
	require.Equal(t, http.StatusOK, publishResp.StatusCode)

	var published api.MoveRequestToPublishResponseWrapper
	require.NoError(t, json.NewDecoder(publishResp.Body).Decode(&published))
	return published.Data
}

func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()
	if err := resp.Body.Close(); err != nil {
		t.Errorf("failed to close response body: %v", err)
	}
}

func TestCreateRequest_Success(t *testing.T) {
	passengerID := uuid.New()
	token := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")

	resp := requestJSON(t, http.MethodPost, "/request/", token, api.CreateRequestRequest{
		FromPoint:     "Moscow",
		ToPoint:       "Kazan",
		DepartureTime: time.Now().Add(24 * time.Hour).Format("2006-01-02 15:04"),
		Seats:         2,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var response api.CreateRequestResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	require.Equal(t, "Success", response.Status)
	require.NotEmpty(t, response.Data.ID)
	require.Equal(t, passengerID.String(), response.Data.PassengerID)
	require.Equal(t, "Moscow", response.Data.FromPoint)
	require.Equal(t, "Kazan", response.Data.ToPoint)
	require.Equal(t, 2, response.Data.Seats)
	require.Equal(t, "draft", response.Data.Status)
	require.Nil(t, response.Data.TripID)

	requestID := uuid.MustParse(response.Data.ID)
	t.Cleanup(func() {
		_ = CleanupRequest(context.Background(), testPool, requestID)
	})

	eventType, eventStatus, err := latestOutboxEvent(context.Background(), testPool, requestID)
	require.NoError(t, err)
	assert.Equal(t, "request_created", eventType)
	assert.Equal(t, "pending", eventStatus)
}

func TestMoveRequestToPublish_Success(t *testing.T) {
	passengerID := uuid.New()
	data := createPublishedRequest(t, passengerID, 2)
	requestID := uuid.MustParse(data.ID)
	t.Cleanup(func() {
		_ = CleanupRequest(context.Background(), testPool, requestID)
	})

	require.Equal(t, "published", data.Status)
	eventType, eventStatus, err := latestOutboxEvent(context.Background(), testPool, requestID)
	require.NoError(t, err)
	assert.Equal(t, "request_published", eventType)
	assert.Equal(t, "pending", eventStatus)
}

func TestMoveRequestToAgreed_SuccessAndReattach(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip1, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	trip2, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, trip1)
		_ = CleanupTestData(ctx, testPool, trip2)
	})

	passengerID := uuid.New()
	token := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() {
		_ = CleanupRequest(ctx, testPool, requestID)
	})

	resp := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", token, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip1.TripID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var attached api.MoveRequestToAgreedResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&attached))
	require.Equal(t, "pre_agreed", attached.Data.Status)
	require.Equal(t, trip1.TripID.String(), *attached.Data.TripID)

	eventType, _, err := latestOutboxEvent(ctx, testPool, requestID)
	require.NoError(t, err)
	assert.Equal(t, "request_preconfirmed", eventType)

	payloadBytes, err := outboxPayloadByType(ctx, testPool, requestID, "request_preconfirmed")
	require.NoError(t, err)
	var preconfirmed events.RequestEvent
	require.NoError(t, json.Unmarshal(payloadBytes, &preconfirmed))
	assert.Equal(t, driverID.String(), preconfirmed.DriverID)
	assert.Equal(t, passengerID.String(), preconfirmed.PassengerID)
	assert.Equal(t, trip1.TripID.String(), preconfirmed.TripID)

	resp2 := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", token, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip2.TripID,
	})
	defer closeBody(t, resp2)
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var reattached api.MoveRequestToAgreedResponseWrapper
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&reattached))
	require.Equal(t, "pre_agreed", reattached.Data.Status)
	require.Equal(t, trip2.TripID.String(), *reattached.Data.TripID)
}

func TestMoveRequestToAgreed_TripNotPublished(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTrip(ctx, testPool, driverID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	token := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	resp := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", token, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestMoveRequestToAgreed_NotEnoughSeats(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	token := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	reqData := createPublishedRequest(t, passengerID, 8)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	resp := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", token, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	getResp := requestJSON(t, http.MethodGet, "/request/"+requestID.String(), token, nil)
	defer closeBody(t, getResp)
	var got api.GetRequestResponseWrapper
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	require.Equal(t, "published", got.Data.Status)
	require.Nil(t, got.Data.TripID)
}

func TestConfirmRequest_DriverTrueAndFalse(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	driverToken := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	attach := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", passengerToken, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	closeBody(t, attach)

	confirmTrue := true
	resp := requestJSON(t, http.MethodPut, "/request/confirm", driverToken, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmTrue,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var confirmed api.ConfirmRequestResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&confirmed))
	require.Equal(t, "agreed", confirmed.Data.Status)
	require.Equal(t, trip.TripID.String(), *confirmed.Data.TripID)

	eventType, _, err := latestOutboxEvent(ctx, testPool, requestID)
	require.NoError(t, err)
	assert.Equal(t, "request_confirmed", eventType)

	payloadBytes, err := outboxPayloadByType(ctx, testPool, requestID, "request_confirmed")
	require.NoError(t, err)
	var confirmedEvent events.RequestEvent
	require.NoError(t, json.Unmarshal(payloadBytes, &confirmedEvent))
	assert.Equal(t, passengerID.String(), confirmedEvent.PassengerID)
	assert.Equal(t, driverID.String(), confirmedEvent.DriverID)
}

func TestConfirmRequest_DriverReject(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	driverToken := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	attach := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", passengerToken, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	closeBody(t, attach)

	confirmFalse := false
	resp := requestJSON(t, http.MethodPut, "/request/confirm", driverToken, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmFalse,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rejected api.ConfirmRequestResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&rejected))
	require.Equal(t, "published", rejected.Data.Status)
	require.Nil(t, rejected.Data.TripID)

	eventType, _, err := latestOutboxEvent(ctx, testPool, requestID)
	require.NoError(t, err)
	assert.Equal(t, "request_rejected", eventType)

	payloadBytes, err := outboxPayloadByType(ctx, testPool, requestID, "request_rejected")
	require.NoError(t, err)
	var rejectedEvent events.RequestEvent
	require.NoError(t, json.Unmarshal(payloadBytes, &rejectedEvent))
	assert.Equal(t, passengerID.String(), rejectedEvent.PassengerID)
}

func TestConfirmRequest_PassengerReject(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	attach := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", passengerToken, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	closeBody(t, attach)

	confirmFalse := false
	resp := requestJSON(t, http.MethodPut, "/request/confirm", passengerToken, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmFalse,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var rejected api.ConfirmRequestResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&rejected))
	require.Equal(t, "published", rejected.Data.Status)
	require.Nil(t, rejected.Data.TripID)
}

func TestConfirmRequest_PassengerCannotConfirm(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	attach := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", passengerToken, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	closeBody(t, attach)

	confirmTrue := true
	resp := requestJSON(t, http.MethodPut, "/request/confirm", passengerToken, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmTrue,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestConfirmRequest_ForeignDriverForbidden(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	otherDriverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	otherDriverToken := testutils.GenerateTestToken(otherDriverID.String(), "other", "o@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	attach := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", passengerToken, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	closeBody(t, attach)

	confirmTrue := true
	respTrue := requestJSON(t, http.MethodPut, "/request/confirm", otherDriverToken, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmTrue,
	})
	defer closeBody(t, respTrue)
	require.Equal(t, http.StatusForbidden, respTrue.StatusCode)

	confirmFalse := false
	respFalse := requestJSON(t, http.MethodPut, "/request/confirm", otherDriverToken, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmFalse,
	})
	defer closeBody(t, respFalse)
	require.Equal(t, http.StatusForbidden, respFalse.StatusCode)
}

func TestConfirmRequest_WithoutDriverPermission(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passengerID := uuid.New()
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	reqData := createPublishedRequest(t, passengerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(ctx, testPool, requestID) })

	attach := requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", passengerToken, api.MoveRequestToAgreedRequest{
		RequestID: requestID,
		TripID:    trip.TripID,
	})
	closeBody(t, attach)

	token := testutils.GenerateTestTokenWithOptions(testutils.TestTokenOptions{
		UserID:      driverID.String(),
		Username:    "nodriver",
		Email:       "n@example.com",
		Roles:       []string{"client"},
		Permissions: []string{},
	})
	confirmTrue := true
	resp := requestJSON(t, http.MethodPut, "/request/confirm", token, api.ConfirmRequestRequest{
		RequestID: requestID,
		Confirm:   &confirmTrue,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestPassengerCannotTouchForeignRequest(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	reqData := createPublishedRequest(t, ownerID, 1)
	requestID := uuid.MustParse(reqData.ID)
	t.Cleanup(func() { _ = CleanupRequest(context.Background(), testPool, requestID) })

	otherToken := testutils.GeneratePassengerToken(otherID.String(), "other", "o@example.com")
	resp := requestJSON(t, http.MethodPut, "/request/move_to_publish", otherToken, api.MoveRequestToPublishRequest{
		RequestID: requestID,
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestConfirmRequest_SeatsOverflowOnSecondAgree(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	trip, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() { _ = CleanupTestData(ctx, testPool, trip) })

	passenger1 := uuid.New()
	passenger2 := uuid.New()
	token1 := testutils.GeneratePassengerToken(passenger1.String(), "p1", "p1@example.com")
	token2 := testutils.GeneratePassengerToken(passenger2.String(), "p2", "p2@example.com")
	driverToken := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")

	req1 := createPublishedRequest(t, passenger1, 3)
	req2 := createPublishedRequest(t, passenger2, 2)
	id1 := uuid.MustParse(req1.ID)
	id2 := uuid.MustParse(req2.ID)
	t.Cleanup(func() {
		_ = CleanupRequest(ctx, testPool, id1)
		_ = CleanupRequest(ctx, testPool, id2)
	})

	closeBody(t, requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", token1, api.MoveRequestToAgreedRequest{
		RequestID: id1, TripID: trip.TripID,
	}))
	closeBody(t, requestJSON(t, http.MethodPut, "/request/move_request_to_agreed", token2, api.MoveRequestToAgreedRequest{
		RequestID: id2, TripID: trip.TripID,
	}))

	confirmTrue := true
	first := requestJSON(t, http.MethodPut, "/request/confirm", driverToken, api.ConfirmRequestRequest{
		RequestID: id1, Confirm: &confirmTrue,
	})
	defer closeBody(t, first)
	require.Equal(t, http.StatusOK, first.StatusCode)

	second := requestJSON(t, http.MethodPut, "/request/confirm", driverToken, api.ConfirmRequestRequest{
		RequestID: id2, Confirm: &confirmTrue,
	})
	defer closeBody(t, second)
	require.Equal(t, http.StatusConflict, second.StatusCode)
}

func TestMoveRequestToStarted_Removed(t *testing.T) {
	passengerID := uuid.New()
	token := testutils.GeneratePassengerToken(passengerID.String(), "passenger", "p@example.com")
	resp := requestJSON(t, http.MethodPut, "/request/move_to_started", token, map[string]any{
		"requestId": uuid.New().String(),
	})
	defer closeBody(t, resp)
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
