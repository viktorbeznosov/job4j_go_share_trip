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

type MoveRequestToAgreedRequest struct {
	RequestID uuid.UUID `json:"requestId"`
	TripID    uuid.UUID `json:"tripId"`
}

func (h *RequestHandler) MoveRequestToAgreed(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger := logctx.Logger(ctx).With(slog.String("handler", "MoveRequestToAgreed"))

	var req MoveRequestToAgreedRequest
	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}
	if req.RequestID == uuid.Nil {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrRequestIDRequired)
	}
	if !validators.IsValidUUID(req.RequestID.String()) {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrInvalidRequestID)
	}
	if req.TripID == uuid.Nil {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrTripIDRequired)
	}
	if !validators.IsValidUUID(req.TripID.String()) {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrInvalidTripID)
	}

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}
	passengerID, err := parseClientID(claims.Subject)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get passenger id")
	}

	serviceResp, err := h.RequestService.MoveToAgreed(ctx, req.RequestID, passengerID, req.TripID)
	if err != nil {
		logger.Warn("Failed to attach request", slog.Any("error", err))
		if errors.Is(err, tripErrors.ErrRequestNotFound) || errors.Is(err, tripErrors.ErrTripNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		if errors.Is(err, tripErrors.ErrPassengerNotOwner) {
			return h.errorMapper.MapForbidden(c, err)
		}
		if errors.Is(err, tripErrors.ErrRequestNotPublished) ||
			errors.Is(err, tripErrors.ErrTripNotPublished) ||
			errors.Is(err, tripErrors.ErrRequestNotEnoughSeats) {
			return h.errorMapper.MapConflict(c, err)
		}
		return h.errorMapper.MapError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data:   newRequestResponse(*serviceResp),
	})
}
