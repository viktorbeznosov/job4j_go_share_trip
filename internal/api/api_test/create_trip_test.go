package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"job4j_go_share_trip/internal/api"
	testutils "job4j_go_share_trip/internal/test_utils"
)

func Test_CreateTrip(t *testing.T) {
	t.Run("success - создание поездки", func(t *testing.T) {
		payload := api.CreateTripRequest{
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

		userID := uuid.New()

		token := testutils.GenerateTestToken(
			userID.String(),
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

		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var response api.CreateTripResponseWrapper
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		expected := api.CreateTripResponse{
			ID:            response.Data.ID, // ID генерируется на сервере
			DriverID:      userID.String(),
			FromPoint:     payload.FromPoint,
			ToPoint:       payload.ToPoint,
			Seats:         payload.Seats,
			Status:        "draft",
			CreatedAt:     response.Data.CreatedAt,     // время генерируется на сервере
			DepartureTime: response.Data.DepartureTime, // парсится на сервере
		}

		assert.Equal(t, expected, response.Data)
	})
}