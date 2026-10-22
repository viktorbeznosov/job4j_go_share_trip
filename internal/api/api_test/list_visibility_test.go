package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"job4j_go_share_trip/internal/api"
	requestEntity "job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/business/trip/entity"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func TestListAndGetTripsVisibility(t *testing.T) {
	ctx := context.Background()
	driverA := uuid.New()
	driverB := uuid.New()
	passengerID := uuid.New()

	ownDraft, err := CreateTestTripWithStatus(ctx, testPool, driverA, entity.StatusDraft)
	require.NoError(t, err)
	ownPublished, err := CreateTestTripWithStatus(ctx, testPool, driverA, entity.StatusPublished)
	require.NoError(t, err)
	foreignPublished, err := CreateTestTripWithStatus(ctx, testPool, driverB, entity.StatusPublished)
	require.NoError(t, err)
	foreignDraft, err := CreateTestTripWithStatus(ctx, testPool, driverB, entity.StatusDraft)
	require.NoError(t, err)
	linkedDraft, err := CreateTestTripWithStatus(ctx, testPool, driverB, entity.StatusDraft)
	require.NoError(t, err)
	preapprovedA, err := CreateTestTripWithStatus(ctx, testPool, driverA, entity.StatusPreapproved)
	require.NoError(t, err)
	preapprovedB, err := CreateTestTripWithStatus(ctx, testPool, driverB, entity.StatusPreapproved)
	require.NoError(t, err)

	linkedReq, err := InsertTestRequest(ctx, testPool, passengerID, requestEntity.StatusPreAgreed, &linkedDraft.TripID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = CleanupRequest(ctx, testPool, linkedReq.ID)
		_ = CleanupTestData(ctx, testPool, ownDraft)
		_ = CleanupTestData(ctx, testPool, ownPublished)
		_ = CleanupTestData(ctx, testPool, foreignPublished)
		_ = CleanupTestData(ctx, testPool, foreignDraft)
		_ = CleanupTestData(ctx, testPool, linkedDraft)
		_ = CleanupTestData(ctx, testPool, preapprovedA)
		_ = CleanupTestData(ctx, testPool, preapprovedB)
	})

	driverToken := testutils.GenerateTestToken(driverA.String(), "driver", "d@example.com")
	passengerToken := testutils.GeneratePassengerToken(passengerID.String(), "p", "p@example.com")
	directorToken := testutils.GenerateDirectorToken(uuid.New().String(), "director", "dir@example.com")
	adminToken := testutils.GenerateAdminToken(uuid.New().String(), "admin", "a@example.com")

	driverList := listTrips(t, driverToken, "")
	require.True(t, containsTrip(driverList.Data.Items, ownDraft.TripID))
	require.True(t, containsTrip(driverList.Data.Items, ownPublished.TripID))
	require.False(t, containsTrip(driverList.Data.Items, foreignPublished.TripID))
	require.False(t, containsTrip(driverList.Data.Items, foreignDraft.TripID))

	require.Equal(t, http.StatusOK, getTripStatus(t, driverToken, ownDraft.TripID))
	require.Equal(t, http.StatusNotFound, getTripStatus(t, driverToken, foreignPublished.TripID))
	require.Equal(t, http.StatusNotFound, getTripStatus(t, driverToken, foreignDraft.TripID))

	passengerList := listTrips(t, passengerToken, "")
	require.True(t, containsTrip(passengerList.Data.Items, ownPublished.TripID))
	require.True(t, containsTrip(passengerList.Data.Items, foreignPublished.TripID))
	require.True(t, containsTrip(passengerList.Data.Items, linkedDraft.TripID))
	require.False(t, containsTrip(passengerList.Data.Items, foreignDraft.TripID))
	require.False(t, containsTrip(passengerList.Data.Items, ownDraft.TripID))

	require.Equal(t, http.StatusOK, getTripStatus(t, passengerToken, foreignPublished.TripID))
	require.Equal(t, http.StatusOK, getTripStatus(t, passengerToken, linkedDraft.TripID))
	require.Equal(t, http.StatusNotFound, getTripStatus(t, passengerToken, foreignDraft.TripID))

	directorList := listTrips(t, directorToken, "")
	require.True(t, containsTrip(directorList.Data.Items, preapprovedA.TripID))
	require.True(t, containsTrip(directorList.Data.Items, preapprovedB.TripID))
	require.False(t, containsTrip(directorList.Data.Items, ownDraft.TripID))
	require.False(t, containsTrip(directorList.Data.Items, ownPublished.TripID))
	require.False(t, containsTrip(directorList.Data.Items, foreignDraft.TripID))
	require.False(t, containsTrip(directorList.Data.Items, foreignPublished.TripID))
	require.Equal(t, http.StatusOK, getTripStatus(t, directorToken, preapprovedA.TripID))
	require.Equal(t, http.StatusOK, getTripStatus(t, directorToken, preapprovedB.TripID))
	require.Equal(t, http.StatusNotFound, getTripStatus(t, directorToken, ownDraft.TripID))
	require.Equal(t, http.StatusNotFound, getTripStatus(t, directorToken, foreignPublished.TripID))

	directorFilterForbidden := requestJSON(t, http.MethodGet, "/trip/?driverId="+driverA.String(), directorToken, nil)
	defer closeBody(t, directorFilterForbidden)
	require.Equal(t, http.StatusBadRequest, directorFilterForbidden.StatusCode)

	adminList := listTrips(t, adminToken, "")
	require.True(t, containsTrip(adminList.Data.Items, ownDraft.TripID))
	require.True(t, containsTrip(adminList.Data.Items, foreignDraft.TripID))
	require.Equal(t, http.StatusOK, getTripStatus(t, adminToken, foreignDraft.TripID))

	filtered := listTrips(t, adminToken, "driverId="+driverA.String())
	require.True(t, containsTrip(filtered.Data.Items, ownDraft.TripID))
	require.False(t, containsTrip(filtered.Data.Items, foreignPublished.TripID))

	driverFilterForbidden := requestJSON(t, http.MethodGet, "/trip/?driverId="+driverB.String(), driverToken, nil)
	defer closeBody(t, driverFilterForbidden)
	require.Equal(t, http.StatusBadRequest, driverFilterForbidden.StatusCode)

	badSort := requestJSON(t, http.MethodGet, "/trip/?sortBy=seats", driverToken, nil)
	defer closeBody(t, badSort)
	require.Equal(t, http.StatusBadRequest, badSort.StatusCode)

	statusFilter := listTrips(t, adminToken, "status=published&fromPoint=Moscow")
	require.GreaterOrEqual(t, statusFilter.Data.Total, 1)
	for _, item := range statusFilter.Data.Items {
		require.Equal(t, "published", item.Status)
		require.Equal(t, "Moscow", item.FromPoint)
	}

	missing := getTripStatus(t, adminToken, uuid.New())
	require.Equal(t, http.StatusNotFound, missing)
}

func TestListAndGetRequestsVisibility(t *testing.T) {
	ctx := context.Background()
	driverA := uuid.New()
	driverB := uuid.New()
	passengerA := uuid.New()
	passengerB := uuid.New()

	tripA, err := CreateTestTripWithStatus(ctx, testPool, driverA, entity.StatusPublished)
	require.NoError(t, err)
	tripB, err := CreateTestTripWithStatus(ctx, testPool, driverB, entity.StatusPublished)
	require.NoError(t, err)

	ownDraft, err := InsertTestRequest(ctx, testPool, passengerA, requestEntity.StatusDraft, nil)
	require.NoError(t, err)
	ownPublished, err := InsertTestRequest(ctx, testPool, passengerA, requestEntity.StatusPublished, nil)
	require.NoError(t, err)
	ownPreAgreed, err := InsertTestRequest(ctx, testPool, passengerA, requestEntity.StatusPreAgreed, &tripA.TripID)
	require.NoError(t, err)
	foreignPublished, err := InsertTestRequest(ctx, testPool, passengerB, requestEntity.StatusPublished, nil)
	require.NoError(t, err)
	foreignPreAgreed, err := InsertTestRequest(ctx, testPool, passengerB, requestEntity.StatusPreAgreed, &tripB.TripID)
	require.NoError(t, err)
	onMyTrip, err := InsertTestRequest(ctx, testPool, passengerB, requestEntity.StatusPreAgreed, &tripA.TripID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = CleanupRequest(ctx, testPool, ownDraft.ID)
		_ = CleanupRequest(ctx, testPool, ownPublished.ID)
		_ = CleanupRequest(ctx, testPool, ownPreAgreed.ID)
		_ = CleanupRequest(ctx, testPool, foreignPublished.ID)
		_ = CleanupRequest(ctx, testPool, foreignPreAgreed.ID)
		_ = CleanupRequest(ctx, testPool, onMyTrip.ID)
		_ = CleanupTestData(ctx, testPool, tripA)
		_ = CleanupTestData(ctx, testPool, tripB)
	})

	passengerToken := testutils.GeneratePassengerToken(passengerA.String(), "p", "p@example.com")
	driverToken := testutils.GenerateTestToken(driverA.String(), "driver", "d@example.com")
	directorToken := testutils.GenerateDirectorToken(uuid.New().String(), "director", "dir@example.com")
	adminToken := testutils.GenerateAdminToken(uuid.New().String(), "admin", "a@example.com")

	directorListForbidden := requestJSON(t, http.MethodGet, "/request/", directorToken, nil)
	defer closeBody(t, directorListForbidden)
	require.Equal(t, http.StatusForbidden, directorListForbidden.StatusCode)
	require.Equal(t, http.StatusForbidden, getRequestStatus(t, directorToken, ownDraft.ID))

	passengerList := listRequests(t, passengerToken, "")
	require.True(t, containsRequest(passengerList.Data.Items, ownDraft.ID))
	require.True(t, containsRequest(passengerList.Data.Items, ownPublished.ID))
	require.False(t, containsRequest(passengerList.Data.Items, foreignPublished.ID))
	require.Equal(t, http.StatusOK, getRequestStatus(t, passengerToken, ownDraft.ID))
	require.Equal(t, http.StatusNotFound, getRequestStatus(t, passengerToken, foreignPublished.ID))

	driverList := listRequests(t, driverToken, "")
	require.True(t, containsRequest(driverList.Data.Items, ownPreAgreed.ID))
	require.True(t, containsRequest(driverList.Data.Items, onMyTrip.ID))
	require.False(t, containsRequest(driverList.Data.Items, ownDraft.ID))
	require.False(t, containsRequest(driverList.Data.Items, ownPublished.ID))
	require.False(t, containsRequest(driverList.Data.Items, foreignPreAgreed.ID))
	require.Equal(t, http.StatusOK, getRequestStatus(t, driverToken, onMyTrip.ID))
	require.Equal(t, http.StatusNotFound, getRequestStatus(t, driverToken, ownPublished.ID))
	require.Equal(t, http.StatusNotFound, getRequestStatus(t, driverToken, foreignPreAgreed.ID))

	adminList := listRequests(t, adminToken, "")
	require.True(t, containsRequest(adminList.Data.Items, ownDraft.ID))
	require.True(t, containsRequest(adminList.Data.Items, foreignPreAgreed.ID))
	require.Equal(t, http.StatusOK, getRequestStatus(t, adminToken, foreignPublished.ID))

	byPassenger := listRequests(t, adminToken, "passengerId="+passengerA.String())
	require.True(t, containsRequest(byPassenger.Data.Items, ownDraft.ID))
	require.False(t, containsRequest(byPassenger.Data.Items, foreignPublished.ID))

	forbidden := requestJSON(t, http.MethodGet, "/request/?passengerId="+passengerB.String(), passengerToken, nil)
	defer closeBody(t, forbidden)
	require.Equal(t, http.StatusBadRequest, forbidden.StatusCode)

	empty := listRequests(t, testutils.GeneratePassengerToken(uuid.New().String(), "none", "n@example.com"), "")
	require.Equal(t, 0, empty.Data.Total)
	require.Empty(t, empty.Data.Items)

	require.Equal(t, http.StatusNotFound, getRequestStatus(t, adminToken, uuid.New()))
}

func TestListTripsSortByDepartureTimeDesc(t *testing.T) {
	ctx := context.Background()
	driverID := uuid.New()
	earlier, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	later, err := CreateTestTripWithStatus(ctx, testPool, driverID, entity.StatusPublished)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = CleanupTestData(ctx, testPool, earlier)
		_ = CleanupTestData(ctx, testPool, later)
	})

	_, err = testPool.Exec(ctx, `UPDATE trips SET departure_time = $1 WHERE id = $2`, time.Now().Add(10*time.Hour), earlier.TripID)
	require.NoError(t, err)
	_, err = testPool.Exec(ctx, `UPDATE trips SET departure_time = $1 WHERE id = $2`, time.Now().Add(40*time.Hour), later.TripID)
	require.NoError(t, err)

	token := testutils.GenerateTestToken(driverID.String(), "driver", "d@example.com")
	listed := listTrips(t, token, "sortBy=departureTime&sortDir=desc")
	require.GreaterOrEqual(t, len(listed.Data.Items), 2)
	require.Equal(t, later.TripID.String(), listed.Data.Items[0].ID)
}

func listTrips(t *testing.T, token, query string) api.ListTripsResponseWrapper {
	t.Helper()
	path := "/trip/"
	if query != "" {
		path += "?" + query
	}
	resp := requestJSON(t, http.MethodGet, path, token, nil)
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body api.ListTripsResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func listRequests(t *testing.T, token, query string) api.ListRequestsResponseWrapper {
	t.Helper()
	path := "/request/"
	if query != "" {
		path += "?" + query
	}
	resp := requestJSON(t, http.MethodGet, path, token, nil)
	defer closeBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var body api.ListRequestsResponseWrapper
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func getTripStatus(t *testing.T, token string, id uuid.UUID) int {
	t.Helper()
	resp := requestJSON(t, http.MethodGet, "/trip/"+id.String(), token, nil)
	defer closeBody(t, resp)
	return resp.StatusCode
}

func getRequestStatus(t *testing.T, token string, id uuid.UUID) int {
	t.Helper()
	resp := requestJSON(t, http.MethodGet, "/request/"+id.String(), token, nil)
	defer closeBody(t, resp)
	return resp.StatusCode
}

func containsTrip(items []api.CreateTripResponse, id uuid.UUID) bool {
	for _, item := range items {
		if item.ID == id.String() {
			return true
		}
	}
	return false
}

func containsRequest(items []api.RequestResponse, id uuid.UUID) bool {
	for _, item := range items {
		if item.ID == id.String() {
			return true
		}
	}
	return false
}
