package handler

import (
	"job4j_go_share_trip/internal/domain/trip/handler/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (h *TripHandler) GetTripByUUID(c *fiber.Ctx) error {
    uuidStr := c.Params("uuid")

    id, err := uuid.Parse(uuidStr)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(response.NewErrorResponse(
            err.Error(),
        ))
    }

    trip, err := h.TripService.GetByTripID(c.Context(), id)

    if (err != nil) {
        return c.Status(fiber.StatusNotFound).JSON(response.NewErrorResponse(
            err.Error(),
        ))
    }

    return c.Status(fiber.StatusCreated).JSON(response.NewSuccessResponse(trip))
}

