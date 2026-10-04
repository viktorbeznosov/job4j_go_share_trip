package middleware

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"

	"job4j_go_share_trip/internal/observability/logctx"
)

func TestCorrelationCreatesAndEchoesProcessIDs(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Use(Correlation(slog.New(slog.NewTextHandler(io.Discard, nil))))
	app.Get("/ping", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		require.Equal(t, "req-123", logctx.RequestID(ctx))
		require.Equal(t, "corr-777", logctx.CorrelationID(ctx))
		require.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", logctx.Traceparent(ctx))
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	req.Header.Set(CorrelationIDHeader, "corr-777")
	req.Header.Set(TraceparentHeader, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	require.Equal(t, "req-123", resp.Header.Get(RequestIDHeader))
	require.Equal(t, "corr-777", resp.Header.Get(CorrelationIDHeader))
	require.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", resp.Header.Get(TraceparentHeader))
}

func TestCorrelationGeneratesIDsWhenMissing(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Use(Correlation(slog.New(slog.NewTextHandler(io.Discard, nil))))
	app.Get("/ping", func(c *fiber.Ctx) error {
		ctx := c.UserContext()
		require.NotEmpty(t, logctx.RequestID(ctx))
		require.Equal(t, logctx.RequestID(ctx), logctx.CorrelationID(ctx))
		return c.SendStatus(fiber.StatusNoContent)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/ping", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusNoContent, resp.StatusCode)
	require.NotEmpty(t, resp.Header.Get(RequestIDHeader))
	require.Equal(t, resp.Header.Get(RequestIDHeader), resp.Header.Get(CorrelationIDHeader))
}
