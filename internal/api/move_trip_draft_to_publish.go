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

type MoveTripDraftToPublishModelRequest struct {
	TripID uuid.UUID `json:"tripId"`
}

type MoveTripDraftToPublishResponse struct {
	TripID string `json:"tripId"`
}

func (h *TripHandler) MoveTripDraftToPublish(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("handler", "MoveTripDraftToPublish"),
	)

	tracer := otel.Tracer("trip-api")
	_, span := tracer.Start(ctx, "MoveTripDraftToPublish")
	defer span.End()

	var req MoveTripDraftToPublishModelRequest

	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}

	if err := req.Validate(); err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	getReq := service.GetTripRequest{
		TripID: req.TripID,
	}

	tripResp, err := h.TripService.GetByTripID(ctx, getReq)
	if err != nil {
		if errors.Is(err, tripErrors.ErrTripNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		return h.errorMapper.MapError(c, err)
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

	if tripResp.DriverID != clientUUID {
		logger.Warn("Forbidden: client is not driver",
			slog.String("client_id", clientUUID.String()),
			slog.String("driver_id", tripResp.DriverID.String()),
		)
		return h.errorMapper.MapForbidden(c,
			fmt.Errorf("client %s is not driver of trip %s", clientUUID, req.TripID),
		)
	}

	if tripResp.Status == string(entity.StatusPublished) {
        return c.SendStatus(fiber.StatusNoContent)
	}

    if tripResp.Status != string(entity.StatusDraft) {
        return h.errorMapper.MapConflict(c,
            fmt.Errorf("invalid trip status: expected %s, got %s", entity.StatusDraft, tripResp.Status),
        )
    }

	serviceReq := service.MoveFromDraftToPublishRequest{
		TripID:    req.TripID,
		DriverID:  clientUUID,
		OldStatus: string(entity.StatusDraft),
		NewStatus: string(entity.StatusPublished),
	}

	serviceResp, err := h.TripService.MoveFromDraftToPublish(ctx, serviceReq)
	if err != nil {
		logger.Warn("Failed to update trip", slog.Any("error", err))
		return h.errorMapper.MapError(c, err)
	}

	span.SetAttributes(
		attribute.String("trip_id", serviceResp.ID.String()),
		attribute.String("client_id", serviceResp.DriverID.String()),
		attribute.String("status", string(serviceResp.Status)),
	)

	logger.Info("trip published successfully", slog.String("trip_id", serviceResp.ID.String()))

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data: MoveTripDraftToPublishResponse{
			TripID: serviceResp.ID.String(),
		},
	})
}

func (r *MoveTripDraftToPublishModelRequest) Validate() error {
	if r.TripID == uuid.Nil {
		return tripErrors.ErrTripIDRequired
	}

	if !validators.IsValidUUID(r.TripID.String()) {
		return tripErrors.ErrInvalidTripID
	}

	return nil
}