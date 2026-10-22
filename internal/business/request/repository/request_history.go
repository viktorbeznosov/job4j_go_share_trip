package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (r *RequestRepository) CreateHistoryTx(
	ctx context.Context,
	db Querier,
	requestID uuid.UUID,
	fromStatus *entity.Status,
	toStatus *entity.Status,
) error {
	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "RequestRepository"),
		slog.String("operation", "CreateHistory"),
		slog.String("request_id", requestID.String()),
	)

	if requestID == uuid.Nil {
		return errors.New("request_id is required")
	}
	if toStatus == nil {
		return errors.New("to_status is required")
	}

	fromStatusStr := ""
	if fromStatus != nil {
		fromStatusStr = string(*fromStatus)
	}

	logger = logger.With(
		slog.String("from_status", fromStatusStr),
		slog.String("to_status", string(*toStatus)),
	)

	logger.Info("history create started")

	var (
		builder strings.Builder
		fields  []string
		args    []any
	)

	builder.WriteString("INSERT INTO public.request_history (")

	fields = append(fields, "id")
	args = append(args, uuid.New())

	fields = append(fields, "request_id")
	args = append(args, requestID)

	fields = append(fields, "to_status")
	args = append(args, *toStatus)

	if fromStatus != nil && *fromStatus != "" {
		fields = append(fields, "from_status")
		args = append(args, *fromStatus)
	}

	placeholders := make([]string, len(args))
	for i := range args {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	builder.WriteString(strings.Join(fields, ", "))
	builder.WriteString(") VALUES (")
	builder.WriteString(strings.Join(placeholders, ", "))
	builder.WriteString(")")

	_, err := db.Exec(ctx, builder.String(), args...)
	if err != nil {
		logger.Error("save request history failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec create request history: %w", err)
	}

	logger.Info("save request history completed")
	return nil
}
