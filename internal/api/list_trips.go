package api

import (
	"github.com/gofiber/fiber/v2"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/middleware"
)

type ListTripsResponse struct {
	Items  []CreateTripResponse `json:"items"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
	Total  int                  `json:"total"`
}

func (h *TripHandler) ListTrips(c *fiber.Ctx) error {
	claims, err := middleware.ClaimsFromContext(c)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get claims")
	}

	viewer, err := viewerFromClaims(claims)
	if err != nil {
		return h.errorMapper.MapParseError(c, err, "Error get client id")
	}

	params, err := parseListParams(c)
	if err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	driverID, err := parseOptionalUUIDQuery(c.Query("driverId"), tripErrors.ErrInvalidTripID)
	if err != nil {
		return h.errorMapper.MapValidationError(c, tripErrors.ErrInvalidTripID)
	}

	result, err := h.TripService.List(c.UserContext(), service.ListTripsInput{
		Viewer:     viewer,
		DriverID:   driverID,
		ListParams: params,
	})
	if err != nil {
		return h.errorMapper.MapError(c, err)
	}

	items := make([]CreateTripResponse, 0, len(result.Items))
	for _, trip := range result.Items {
		items = append(items, CreateTripResponse{
			ID:            trip.ID.String(),
			DriverID:      trip.DriverID.String(),
			FromPoint:     trip.FromPoint,
			ToPoint:       trip.ToPoint,
			DepartureTime: trip.DepartureTime,
			Seats:         trip.Seats,
			Status:        trip.Status,
			CreatedAt:     trip.CreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data: ListTripsResponse{
			Items:  items,
			Limit:  result.Limit,
			Offset: result.Offset,
			Total:  result.Total,
		},
	})
}
