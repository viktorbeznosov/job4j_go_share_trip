package logctx

import (
	"context"
	"log/slog"
)

type loggerKey struct{}
type requestIDKey struct{}

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
	if ctx == nil {
		return ""
	}
	value, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return ""
	}
	return value
}
