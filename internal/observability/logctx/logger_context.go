package logctx

import (
	"context"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type loggerKey struct{}
type requestIDKey struct{}
type correlationIDKey struct{}
type traceparentKey struct{}
type tripIDKey struct{}
type userIDKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil || logger == nil {
		return context.WithValue(context.Background(), loggerKey{}, slog.Default())
	}
	return context.WithValue(ctx, loggerKey{}, logger)
}

func Logger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}

	logger, ok := ctx.Value(loggerKey{}).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}
	return logger
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	return stringValue(ctx, requestIDKey{})
}

func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, correlationIDKey{}, correlationID)
}

func CorrelationID(ctx context.Context) string {
	return stringValue(ctx, correlationIDKey{})
}

func WithTraceparent(ctx context.Context, traceparent string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, traceparentKey{}, traceparent)
}

func Traceparent(ctx context.Context) string {
	if value := stringValue(ctx, traceparentKey{}); value != "" {
		return value
	}
	return TraceparentFromSpan(ctx)
}

func TraceparentFromSpan(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

func TraceID(ctx context.Context) string {
	return TraceIDFromParent(Traceparent(ctx))
}

func TraceIDFromParent(traceparent string) string {
	parts := strings.Split(traceparent, "-")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func WithTripID(ctx context.Context, tripID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, tripIDKey{}, tripID)
}

func TripID(ctx context.Context) string {
	return stringValue(ctx, tripIDKey{})
}

func WithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userIDKey{}, userID)
}

func UserID(ctx context.Context) string {
	return stringValue(ctx, userIDKey{})
}

func stringValue(ctx context.Context, key any) string {
	if ctx == nil {
		return ""
	}
	value, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return value
}
