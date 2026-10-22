package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/middleware"
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

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}

	viewer, err := viewerFromClaims(claims)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get client id")
	}

	tripResponse, err := h.TripService.GetByTripIDForViewer(c.UserContext(), id, viewer)
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
		Data: CreateTripResponse{
			ID:            tripResponse.ID.String(),
			DriverID:      tripResponse.DriverID.String(),
			FromPoint:     tripResponse.FromPoint,
			ToPoint:       tripResponse.ToPoint,
			DepartureTime: tripResponse.DepartureTime,
			Seats:         tripResponse.Seats,
			Status:        tripResponse.Status,
			CreatedAt:     tripResponse.CreatedAt,
		},
	})
}
