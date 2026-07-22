package handler

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/handler/request"
	"job4j_go_share_trip/internal/domain/trip/handler/response"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (h *TripHandler) MoveTripDraftToPublish(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("handler", "MoveTripDraftToPublish"),
	)

	tracer := otel.Tracer("trip-api")
	_, span := tracer.Start(ctx, "MoveTripDraftToPublish")
	defer span.End()

	var req request.MoveTripDraftToPublishModelRequest

	// 1. Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			"Invalid JSON body",
			err.Error(),
		))
	}

	// 2. Валидируем запрос
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			err.Error(),
		))
	}

	trip, err := h.TripService.GetForUpdateByID(ctx, req.TripID)
	if err != nil {
		logger.Warn("Error get Trip", slog.Any("error", err))
		return c.Status(fiber.StatusNotFound).JSON(response.NewErrorResponse(
			"Trip not found",
			err.Error(),
		))
	}

	// Проверка прав
	if trip.DriverID != req.ClientID {
		logger.Warn("Forbidden: client is not driver",
			slog.String("client_id", req.ClientID.String()),
			slog.String("driver_id", trip.DriverID.String()),
		)
		return c.Status(fiber.StatusForbidden).JSON(response.NewErrorResponse(
			fmt.Sprintf("client %s is not driver of trip %s", req.ClientID, req.TripID),
		))
	}

	// Проверка статуса
	if trip.Status == entity.StatusPublished {
		return c.Status(fiber.StatusNoContent).JSON(response.NewSuccessResponse(
			response.NewMoveTripDraftToPublishResponse(&trip),
		))
	}

	if trip.Status != entity.StatusDraft {
		return c.Status(fiber.StatusConflict).JSON(response.NewErrorResponse(
			fmt.Sprintf("invalid trip status: expected %s, got %s", entity.StatusDraft, trip.Status),
		))
	}

	// Обновляем
	oldStatus := trip.Status
	trip.Status = entity.StatusPublished

	updatedTrip, err := h.TripService.Update(ctx, &trip, oldStatus)
	if err != nil {
		logger.Warn("Failed to update trip", slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(response.NewErrorResponse(
			"Failed to update trip",
		))
	}

	span.SetAttributes(
		attribute.String("trip_id", trip.ID.String()),
		attribute.String("client_id", trip.DriverID.String()),
		attribute.String("status", string(trip.Status)),
	)

	logger.Info("trip published successfully", slog.String("trip_id", trip.ID.String()))

	// ✅ Упрощенный ответ
	return c.Status(fiber.StatusOK).JSON(response.NewSuccessResponse(
		response.NewMoveTripDraftToPublishResponse(updatedTrip),
	))
}