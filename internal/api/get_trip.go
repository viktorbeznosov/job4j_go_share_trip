package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/service"
)

func (h *TripHandler) GetTripByUUID(c *fiber.Ctx) error {
	uuidStr := c.Params("uuid")

	tracer := otel.Tracer("trip-api")

	_, span := tracer.Start(c.UserContext(), "GetTripHandler")
	defer span.End()

	c.Set("trace-id", span.SpanContext().TraceID().String())

	id, err := uuid.Parse(uuidStr)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid UUID format")
	}

	serviceRequest := service.GetTripRequest{
		TripID: id,
	}

	tripResponse, err := h.TripService.GetByTripID(c.Context(), serviceRequest)
	if err != nil {
		if errors.Is(err, tripErrors.ErrTripNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		return h.errorMapper.MapError(c, err)
	}

	span.SetAttributes(
		attribute.String("trip_id", tripResponse.ID.String()),
		attribute.String("client_id", tripResponse.DriverID.String()),
	)

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data:   tripResponse,
	})
}