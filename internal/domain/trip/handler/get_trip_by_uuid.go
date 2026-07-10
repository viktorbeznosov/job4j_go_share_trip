package handler

import (
	"job4j_go_share_trip/internal/domain/trip/handler/response"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func (h *TripHandler) GetTripByUUID(c *fiber.Ctx) error {
    uuidStr := c.Params("uuid")

	tracer := otel.Tracer("trip-api")

	_, span := tracer.Start(c.UserContext(), "GetTripHandler")
	defer span.End()

	c.Set("trace-id", span.SpanContext().TraceID().String())

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

	span.SetAttributes(
		attribute.String("trip_id", trip.ID.String()),
		attribute.String("client_id", trip.DriverID.String()),
	)

    return c.Status(fiber.StatusCreated).JSON(response.NewSuccessResponse(trip))
}

