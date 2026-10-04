package middleware

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j_go_share_trip/internal/observability/logctx"
)

const (
	RequestIDHeader     = "X-Request-ID"
	CorrelationIDHeader = "X-Correlation-ID"
	TraceparentHeader   = "traceparent"
	TripIDHeader        = "X-Trip-ID"
	UserIDHeader        = "X-User-ID"
	LoggerLocalKey      = "logger"
)

func Correlation(baseLogger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := firstNonEmpty(c.Get(RequestIDHeader), c.Get("X-Request-Id"))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		correlationID := firstNonEmpty(c.Get(CorrelationIDHeader), requestID)
		traceparent := firstNonEmpty(c.Get(TraceparentHeader), logctx.TraceparentFromSpan(c.UserContext()))

		c.Set(RequestIDHeader, requestID)
		c.Set(CorrelationIDHeader, correlationID)
		if traceparent != "" {
			c.Set(TraceparentHeader, traceparent)
		}

		requestLogger := baseLogger.With(
			slog.String("service", "sharetrip"),
			slog.String("request_id", requestID),
			slog.String("correlation_id", correlationID),
			slog.String("trace_id", logctx.TraceIDFromParent(traceparent)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
		)

		ctx := c.UserContext()
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = logctx.WithRequestID(ctx, requestID)
		ctx = logctx.WithCorrelationID(ctx, correlationID)
		ctx = logctx.WithTraceparent(ctx, traceparent)
		ctx = logctx.WithLogger(ctx, requestLogger)

		c.SetUserContext(ctx)
		c.Locals(LoggerLocalKey, requestLogger)

		return c.Next()
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
