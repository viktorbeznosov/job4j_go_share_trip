package handler

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/handler/request"
	"job4j_go_share_trip/internal/domain/trip/handler/response"
)

func (h *TripHandler) CreateTrip(c *fiber.Ctx) error {
	var req request.CreateTripRequest

	// 1. Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		log.Printf("JSON parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			"Invalid JSON body",
			err.Error(),
		))
	}

	log.Printf("Received request: %+v", req)

	// 2. Валидируем запрос
	if err := req.Validate(); err != nil {
		log.Printf("Validation error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			err.Error(),
		))
	}

	// 3. Парсим дату
	departureTime, err := req.ParseDepartureTime()
	if err != nil {
		log.Printf("Date parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			err.Error(),
		))
	}

	// 4. Создаем сущность
	trip, err := entity.NewTrip(
		req.DriverID,
		req.FromPoint,
		req.ToPoint,
		departureTime,
		req.Seats,
	)
	if err != nil {
		log.Printf("Failed to create trip entity: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
			err.Error(),
		))
	}

	// 5. Сохраняем в БД
	if err := h.TripService.Create(c.Context(), trip); err != nil {
		log.Printf("Failed to save trip: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(response.NewErrorResponse(
			"Failed to create trip",
		))
	}

	// 6. Успешный ответ с ItemResponse
	return c.Status(fiber.StatusCreated).JSON(response.NewSuccessResponse(trip))
}