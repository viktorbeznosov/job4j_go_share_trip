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

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var response api.Response
		err = json.Unmarshal(respBody, &response)
		require.NoError(t, err)

		require.Equal(t, "Success", response.Status)

		data, ok := response.Data.(map[string]interface{})
		require.True(t, ok)

		require.NotEmpty(t, data["id"])
		require.NotEmpty(t, data["driverId"])
		require.Equal(t, payload.FromPoint, data["fromPoint"])
		require.Equal(t, payload.ToPoint, data["toPoint"])
		require.Equal(t, float64(payload.Seats), data["seats"])
		require.Equal(t, "draft", data["status"])
	})
}