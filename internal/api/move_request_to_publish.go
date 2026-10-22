package api

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/validators"
)

type MoveRequestToPublishRequest struct {
	RequestID uuid.UUID `json:"requestId"`
}

func (h *RequestHandler) MoveRequestToPublish(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger := logctx.Logger(ctx).With(slog.String("handler", "MoveRequestToPublish"))

	var req MoveRequestToPublishRequest
	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}
	if req.RequestID == uuid.Nil {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrRequestIDRequired)
	}
	if !validators.IsValidUUID(req.RequestID.String()) {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrInvalidRequestID)
	}

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}
	passengerID, err := parseClientID(claims.Subject)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get passenger id")
	}

	serviceResp, err := h.RequestService.MoveToPublish(ctx, req.RequestID, passengerID)
	if err != nil {
		logger.Warn("Failed to publish request", slog.Any("error", err))
		if errors.Is(err, tripErrors.ErrRequestNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		if errors.Is(err, tripErrors.ErrPassengerNotOwner) {
			return h.errorMapper.MapForbidden(c, err)
		}
		if errors.Is(err, tripErrors.ErrRequestNotDraft) {
			return h.errorMapper.MapConflict(c, err)
		}
		return h.errorMapper.MapError(c, err)
	}

	if serviceResp.AlreadyPublished {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data:   newRequestResponse(*serviceResp.Request),
	})
}
