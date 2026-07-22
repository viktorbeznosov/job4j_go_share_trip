package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/handler/request"
	"job4j_go_share_trip/internal/domain/trip/handler/response"
	"job4j_go_share_trip/internal/observability/logctx"
)

func (h *TripHandler) CreateTrip(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("handler", "CreateTrip"),
	)

	var req request.CreateTripRequest

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

	// 3. Парсим дату
	departureTime, err := req.ParseDepartureTime()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			err.Error(),
		))
	}

	logger = logger.With(slog.String("client_id", req.DriverID.String()))
	ctx = logctx.WithLogger(ctx, logger)

	// 4. Создаем сущность
	trip, err := entity.NewTrip(
		req.DriverID,
		req.FromPoint,
		req.ToPoint,
		departureTime,
		req.Seats,
	)
	if err != nil {
		logger.Warn("Failed to create trip entity", slog.Any("error", err))
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			err.Error(),
		))
	}

	// 5. Сохраняем в БД
	if err := h.TripService.Create(ctx, trip); err != nil {
		logger.Warn("Failed to save trip", slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(response.NewErrorResponse(
			"Failed to create trip",
		))
	}

	logger.Info("create trip completed", slog.String("trip_id", trip.ID.String()))

	// 6. ✅ Упрощенный ответ
	return c.Status(fiber.StatusCreated).JSON(response.NewSuccessResponse(
		response.NewItemResponse(trip),
	))
}