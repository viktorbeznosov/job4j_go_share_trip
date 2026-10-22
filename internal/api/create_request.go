package api

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/service"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/logctx"
)

type CreateRequestRequest struct {
	FromPoint     string `json:"fromPoint"`
	ToPoint       string `json:"toPoint"`
	DepartureTime string `json:"departureTime"`
	Seats         int    `json:"seats"`
}

func (h *RequestHandler) CreateRequest(c *fiber.Ctx) error {
	ctx := c.UserContext()
	logger := logctx.Logger(ctx).With(slog.String("handler", "CreateRequest"))

	var req CreateRequestRequest
	if err := c.BodyParser(&req); err != nil {
		return h.errorMapper.MapParseError(c, err, "Invalid JSON body")
	}
	if err := req.Validate(); err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	departureTime, err := parseRequestDepartureTime(req.DepartureTime)
	if err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}

	passengerID, err := parseClientID(claims.Subject)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get passenger id")
	}

	logger = logger.With(slog.String("client_id", passengerID.String()))
	ctx = logctx.WithLogger(ctx, logger)

	serviceResp, err := h.RequestService.Create(ctx, service.CreateRequestInput{
		PassengerID:   passengerID,
		FromPoint:     req.FromPoint,
		ToPoint:       req.ToPoint,
		DepartureTime: departureTime,
		Seats:         req.Seats,
	})
	if err != nil {
		logger.Warn("Failed to save request", slog.Any("error", err))
		return h.errorMapper.MapError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(Response{
		Status: "Success",
		Data:   newRequestResponse(*serviceResp),
	})
}

func (r *CreateRequestRequest) Validate() error {
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

func parseRequestDepartureTime(value string) (time.Time, error) {
	departureTime, err := time.Parse("2006-01-02 15:04", value)
	if err != nil {
		return time.Time{}, tripErrors.ErrInvalidDateFormat
	}
	if departureTime.Before(time.Now()) {
		return time.Time{}, tripErrors.ErrDepartureTimePast
	}
	return departureTime, nil
}
