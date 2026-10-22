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

type ConfirmRequestRequest struct {
	RequestID uuid.UUID `json:"requestId"`
	Confirm   *bool     `json:"confirm"`
}

func (h *RequestHandler) ConfirmRequest(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger := logctx.Logger(ctx).With(slog.String("handler", "ConfirmRequest"))

	var req ConfirmRequestRequest
	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}
	if req.RequestID == uuid.Nil {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrRequestIDRequired)
	}
	if !validators.IsValidUUID(req.RequestID.String()) {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrInvalidRequestID)
	}
	if req.Confirm == nil {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrConfirmRequired)
	}

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}
	clientID, err := parseClientID(claims.Subject)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get client id")
	}

	if *req.Confirm && !claims.HasUserPermission(middleware.PermissionShareTripDriver) {
		return h.errorMapper.MapForbidden(c, tripErrors.ErrConfirmRequiresDriver)
	}

	serviceResp, err := h.RequestService.Confirm(ctx, req.RequestID, clientID, *req.Confirm)
	if err != nil {
		logger.Warn("Failed to confirm request", slog.Any("error", err))
		if errors.Is(err, tripErrors.ErrRequestNotFound) || errors.Is(err, tripErrors.ErrTripNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		if errors.Is(err, tripErrors.ErrDriverNotOwner) ||
			errors.Is(err, tripErrors.ErrPassengerNotOwner) ||
			errors.Is(err, tripErrors.ErrRequestConfirmForbidden) ||
			errors.Is(err, tripErrors.ErrConfirmRequiresDriver) {
			return h.errorMapper.MapForbidden(c, err)
		}
		if errors.Is(err, tripErrors.ErrRequestNotPreAgreed) ||
			errors.Is(err, tripErrors.ErrTripNotPublished) ||
			errors.Is(err, tripErrors.ErrRequestNotEnoughSeats) {
			return h.errorMapper.MapConflict(c, err)
		}
		return h.errorMapper.MapError(c, err)
	}

	if serviceResp.AlreadyAgreed {
		return c.SendStatus(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data:   newRequestResponse(*serviceResp.Request),
	})
}
