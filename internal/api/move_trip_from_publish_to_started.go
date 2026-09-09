// internal/api/move_trip_from_publish_to_started.go
package api

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/validators"
)

type MoveTripFromPublishToStartedRequest struct {
	TripID uuid.UUID `json:"tripId"`
}

type MoveTripFromPublishToStartedResponse struct {
	TripID string `json:"tripId"`
}

func (h *TripHandler) MoveTripFromPublishToStarted(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("handler", "MoveTripFromPublishToStarted"),
	)

	tracer := otel.Tracer("trip-api")
	_, span := tracer.Start(ctx, "MoveTripFromPublishToStarted")
	defer span.End()

	var req MoveTripFromPublishToStartedRequest

	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}

	if err := validateMovePublishToStartedRequest(&req); err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		logger.Error("Error get claims", slog.Any("error", err))
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}

	clientUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		logger.Error("Error get driverId", slog.Any("error", err))
		return h.errorMapper.MapParseError(c, err, "Error get driver id")
	}

	getReq := service.GetTripRequest{
		TripID: req.TripID,
	}

	tripResp, err := h.TripService.GetByTripID(ctx, getReq)
	if err != nil {
		if errors.Is(err, tripErrors.ErrTripNotFound) {
			return h.errorMapper.MapNotFound(c, err, "Trip not found")
		}
		return h.errorMapper.MapError(c, err)
	}

	// Проверяем права
	if tripResp.DriverID != clientUUID {
		logger.Warn("Forbidden: client is not driver",
			slog.String("client_id", clientUUID.String()),
			slog.String("driver_id", tripResp.DriverID.String()),
		)
		return h.errorMapper.MapForbidden(c,
			fmt.Errorf("client %s is not driver of trip %s", clientUUID, req.TripID),
			"Client is not the driver of this trip",
		)
	}

	// Если поездка уже начата — 204 No Content
	if tripResp.Status == string(entity.StatusStarted) {
		return c.Status(fiber.StatusNoContent).JSON(Response{
			Status: "Success",
			Data: MoveTripFromPublishToStartedResponse{
				TripID: tripResp.ID.String(),
			},
		})
	}

	// Если статус не published — конфликт
	if tripResp.Status != string(entity.StatusPublished) {
		return h.errorMapper.MapConflict(c,
			fmt.Errorf("invalid trip status: expected %s, got %s", entity.StatusPublished, tripResp.Status),
			fmt.Sprintf("Invalid status: expected %s, got %s", entity.StatusPublished, tripResp.Status),
		)
	}

	serviceReq := service.MoveFromPublishToStartedRequest{
		TripID:    req.TripID,
		DriverID:  clientUUID,
		OldStatus: string(entity.StatusPublished),
		NewStatus: string(entity.StatusStarted),
	}

	serviceResp, err := h.TripService.MoveFromPublishToStarted(ctx, serviceReq)
	if err != nil {
		logger.Warn("Failed to update trip", slog.Any("error", err))
		return h.errorMapper.MapError(c, err)
	}

	span.SetAttributes(
		attribute.String("trip_id", serviceResp.ID.String()),
		attribute.String("client_id", serviceResp.DriverID.String()),
		attribute.String("status", serviceResp.Status),
	)

	logger.Info("trip moved from publish to started successfully", slog.String("trip_id", serviceResp.ID.String()))

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data: MoveTripFromPublishToStartedResponse{
			TripID: serviceResp.ID.String(),
		},
	})
}

func validateMovePublishToStartedRequest(req *MoveTripFromPublishToStartedRequest) error {
	if req.TripID == uuid.Nil {
		return tripErrors.ErrTripIDRequired
	}

	if !validators.IsValidUUID(req.TripID.String()) {
		return tripErrors.ErrInvalidTripID
	}

	return nil
}