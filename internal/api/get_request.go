package api

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/middleware"
)

func (h *RequestHandler) GetRequestByUUID(c *fiber.Ctx) error {
	tracer := otel.Tracer("request-api")
	_, span := tracer.Start(c.UserContext(), "GetRequestHandler")
	defer span.End()

	id, err := uuid.Parse(c.Params("uuid"))
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

	serviceResp, err := h.RequestService.GetByIDForViewer(c.UserContext(), id, viewer)
	if err != nil {
		if errors.Is(err, tripErrors.ErrRequestNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		return h.errorMapper.MapError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data:   newRequestResponse(*serviceResp),
	})
}
