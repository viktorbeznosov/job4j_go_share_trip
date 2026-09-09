// internal/api/create_trip.go
package api

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/logctx"
)

type CreateTripRequest struct {
	FromPoint     string `json:"fromPoint"`
	ToPoint       string `json:"toPoint"`
	DepartureTime string `json:"departureTime"`
	Seats         int    `json:"seats"`
}

type CreateTripResponse struct {
	ID            string    `json:"id"`
	DriverID      string    `json:"driverId"`
	FromPoint     string    `json:"fromPoint"`
	ToPoint       string    `json:"toPoint"`
	DepartureTime time.Time `json:"departureTime"`
	Seats         int       `json:"seats"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (h *TripHandler) CreateTrip(c *fiber.Ctx) error {
	ctx := c.UserContext()

	logger := logctx.Logger(ctx).With(
		slog.String("handler", "CreateTrip"),
	)

	var req CreateTripRequest

	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}

	if err := req.Validate(); err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	departureTime, err := req.ParseDepartureTime()
	if err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		logger.Error("Error get claims", slog.Any("error", err))
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}

	driverUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		logger.Error("Error get driverId", slog.Any("error", err))
		return h.errorMapper.MapParseError(c, err, "Error get driver id")
	}

	logger = logger.With(slog.String("client_id", driverUUID.String()))
	ctx = logctx.WithLogger(ctx, logger)

	serviceReq := service.CreateTripRequest{
		DriverID:      driverUUID,
		FromPoint:     req.FromPoint,
		ToPoint:       req.ToPoint,
		DepartureTime: departureTime,
		Seats:         req.Seats,
	}

	serviceResp, err := h.TripService.Create(ctx, serviceReq)
	if err != nil {
		logger.Warn("Failed to save trip", slog.Any("error", err))
		return h.errorMapper.MapError(c, err)
	}

	logger.Info("create trip completed", slog.String("trip_id", serviceResp.ID.String()))

	tripResponse := CreateTripResponse{
		ID:            serviceResp.ID.String(),
		DriverID:      serviceResp.DriverID.String(),
		FromPoint:     serviceResp.FromPoint,
		ToPoint:       serviceResp.ToPoint,
		DepartureTime: serviceResp.DepartureTime,
		Seats:         serviceResp.Seats,
		Status:        string(serviceResp.Status),
		CreatedAt:     serviceResp.CreatedAt,
	}

	return c.Status(fiber.StatusCreated).JSON(Response{
		Status: "Success",
		Data:   tripResponse,
	})
}

func (r *CreateTripRequest) Validate() error {
	if r.FromPoint == "" {
		return tripErrors.ErrFromPointRequired
	}

	if r.ToPoint == "" {
		return tripErrors.ErrToPointRequired
	}

	if r.DepartureTime == "" {
		return tripErrors.ErrDepartureTimeRequired
	}

	if r.Seats <= 0 {
		return tripErrors.ErrInvalidSeats
	}
	if r.Seats > 10 {
		return tripErrors.ErrSeatsTooHigh
	}

	return nil
}

func (r *CreateTripRequest) ParseDepartureTime() (time.Time, error) {
	layout := "2006-01-02 15:04"

	departureTime, err := time.Parse(layout, r.DepartureTime)
	if err != nil {
		return time.Time{}, tripErrors.ErrInvalidDateFormat
	}

	if departureTime.Before(time.Now()) {
		return time.Time{}, tripErrors.ErrDepartureTimePast
	}

	return departureTime, nil
}