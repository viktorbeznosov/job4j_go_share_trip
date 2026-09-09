package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/observability/logctx"
)


func (r *TripRepository) CreateHistoryTx(
	ctx context.Context,
	db Querier,
	tripID uuid.UUID,
	fromStatus *entity.Status,
	toStatus *entity.Status,
) error {
	logger := logctx.Logger(ctx).With(
		slog.String("layer", "repository"),
		slog.String("repository", "TripRepository"),
		slog.String("operation", "CreateHistory"),
		slog.String("trip_id", tripID.String()),
	)

	if tripID == uuid.Nil {
		return errors.New("trip_id is required")
	}
	if toStatus == nil {
		return errors.New("to_status is required")
	}

	fromStatusStr := ""
	if fromStatus != nil {
		fromStatusStr = string(*fromStatus)
	}
	toStatusStr := string(*toStatus)

	logger = logger.With(
		slog.String("from_status", fromStatusStr),
		slog.String("to_status", toStatusStr),
	)

	logger.Info("history create started")

	var (
		builder strings.Builder
		fields  []string
		args    []any
	)

	builder.WriteString("INSERT INTO public.trip_history (")

	fields = append(fields, "id")
	args = append(args, uuid.New())

	fields = append(fields, "trip_id")
	args = append(args, tripID)

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

	logger.Debug("history query", slog.String("query", builder.String()))

	_, err := db.Exec(ctx, builder.String(), args...)
	if err != nil {
		logger.Error("save trip history failed", slog.Any("error", err))
		return fmt.Errorf("tx.Exec create history: %w", err)
	}

	logger.Info("save trip history completed")
	return nil
}