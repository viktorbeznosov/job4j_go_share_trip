package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	trip_request "job4j_go_share_trip/internal/domain/trip/handler/request"
	create_trip_response "job4j_go_share_trip/internal/domain/trip/handler/response"
)

func Test_CreateTrip(t *testing.T) {
	t.Run("success - создание поездки", func(t *testing.T) {
		payload := trip_request.CreateTripRequest{
			DriverID:      uuid.New(),
			FromPoint:     "TestFromPoint",
			ToPoint:       "TestToPoint",
			DepartureTime: time.Now().AddDate(0, 0, 1).Format("2006-01-02 15:04"),
			Seats:         2,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(
			http.MethodPost,
			"/trip",
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

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var got create_trip_response.ItemResponse
		err = json.Unmarshal(respBody, &got)
		require.NoError(t, err)

		require.Equal(t, create_trip_response.ItemResponse{
			ID:            got.ID,
			DriverID:      got.DriverID,
			FromPoint:     got.FromPoint,
			ToPoint:       got.ToPoint,
			DepartureTime: got.DepartureTime,
			Seats:         got.Seats,
			Status:        got.Status,
		}, got)
	})
}