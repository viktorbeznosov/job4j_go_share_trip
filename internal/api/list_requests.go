package api

import (
	"github.com/gofiber/fiber/v2"

	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/service"
	"job4j_go_share_trip/internal/middleware"
)

type ListRequestsResponse struct {
	Items  []RequestResponse `json:"items"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Total  int               `json:"total"`
}

func (h *RequestHandler) ListRequests(c *fiber.Ctx) error {
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

	passengerID, err := parseOptionalUUIDQuery(c.Query("passengerId"), tripErrors.ErrInvalidPassengerID)
	if err != nil {
		return h.errorMapper.MapValidationError(c, err)
	}

	result, err := h.RequestService.List(c.UserContext(), service.ListRequestsInput{
		Viewer:      viewer,
		PassengerID: passengerID,
		ListParams:  params,
	})
	if err != nil {
		return h.errorMapper.MapError(c, err)
	}

	items := make([]RequestResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, newRequestResponse(item))
	}

	return c.Status(fiber.StatusOK).JSON(Response{
		Status: "Success",
		Data: ListRequestsResponse{
			Items:  items,
			Limit:  result.Limit,
			Offset: result.Offset,
			Total:  result.Total,
		},
	})
}
