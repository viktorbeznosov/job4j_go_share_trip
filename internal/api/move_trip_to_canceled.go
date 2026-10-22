package api

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/logctx"
	"job4j_go_share_trip/internal/validators"
)

type MoveTripToCanceledRequest struct {
	TripID    uuid.UUID `json:"tripId"`
	CompanyID uuid.UUID `json:"companyId"`
}

type MoveTripToCanceledResponse struct {
	ID            string    `json:"id"`
	DriverID      string    `json:"driverId"`
	FromPoint     string    `json:"fromPoint"`
	ToPoint       string    `json:"toPoint"`
	DepartureTime time.Time `json:"departureTime"`
	Seats         int       `json:"seats"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (h *TripHandler) MoveTripToCanceled(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("handler", "MoveTripToCanceled"),
		slog.String("operation", "CancelTrip"),
	)

	logger.Info("cancel trip started", slog.String("result", "started"))

	tracer := otel.Tracer("trip-api")
	_, span := tracer.Start(ctx, "MoveTripToCanceled")
	defer span.End()

	var req MoveTripToCanceledRequest
	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}

	if err := validateMoveTripToCanceledRequest(&req); err != nil {
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

	serviceResp, err := h.TripService.MoveToCanceled(ctx, service.MoveTripToCanceledRequest{
		TripID:    req.TripID,
		ClientID:  clientUUID,
		CompanyID: req.CompanyID,
	})
	if err != nil {
		logger.Warn("Failed to cancel trip", slog.Any("error", err))
		if errors.Is(err, tripErrors.ErrTripNotFound) {
			return h.errorMapper.MapNotFound(c, err)
		}
		return h.errorMapper.MapError(c, err)
	}

	span.SetAttributes(
		attribute.String("trip_id", serviceResp.ID.String()),
		attribute.String("client_id", serviceResp.DriverID.String()),
		attribute.String("status", serviceResp.Status),
	)

	logger.Info("trip moved to canceled successfully", slog.String("trip_id", serviceResp.ID.String()))

	status := fiber.StatusOK
	if serviceResp.AlreadyCanceled {
		status = fiber.StatusNoContent
	}

	return c.Status(status).JSON(Response{
		Status: "Success",
		Data:   newMoveTripToCanceledResponse(*serviceResp),
	})
}

func newMoveTripToCanceledResponse(trip service.MoveTripToCanceledResponse) MoveTripToCanceledResponse {
	return MoveTripToCanceledResponse{
		ID:            trip.ID.String(),
		DriverID:      trip.DriverID.String(),
		FromPoint:     trip.FromPoint,
		ToPoint:       trip.ToPoint,
		DepartureTime: trip.DepartureTime,
		Seats:         trip.Seats,
		Status:        trip.Status,
		CreatedAt:     trip.CreatedAt,
	}
}

func validateMoveTripToCanceledRequest(req *MoveTripToCanceledRequest) error {
	if req.TripID == uuid.Nil {
		return tripErrors.ErrTripIDRequired
	}
	if !validators.IsValidUUID(req.TripID.String()) {
		return tripErrors.ErrInvalidTripID
	}
	if req.CompanyID == uuid.Nil {
		return tripErrors.ErrCompanyIDRequired
	}
	if !validators.IsValidUUID(req.CompanyID.String()) {
		return tripErrors.ErrInvalidCompanyID
	}
	return nil
}
