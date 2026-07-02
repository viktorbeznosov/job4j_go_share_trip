package handler

import (
	"fmt"
	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/handler/request"
	"job4j_go_share_trip/internal/domain/trip/handler/response"
	"log"

	"github.com/gofiber/fiber/v2"
)

func (u *TripHandler) MoveTripDraftToPublish(c *fiber.Ctx) error {
    var req request.MoveTripDraftToPublishModelRequest

	// 1. Парсим JSON
	if err := c.BodyParser(&req); err != nil {
		log.Printf("JSON parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(response.NewMoveTripDraftToPublishErrorResponse(
			"Invalid JSON body",
			err.Error(),
		))
	}

	// 2. Валидируем запрос
	if err := req.Validate(); err != nil {
		log.Printf("Validation error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(response.NewMoveTripDraftToPublishErrorResponse(
			err.Error(),
		))
	}

	trip, err := u.TripService.GetForUpdateByID(c.Context(), req.TripID)
	if err != nil {
        return c.Status(fiber.StatusNotFound).JSON(response.NewMoveTripDraftToPublishErrorResponse(
            "Error get Trip",
            err.Error(),
        ))
	}

	if trip.DriverID != req.ClientID {
        return c.Status(fiber.StatusForbidden).JSON(response.NewMoveTripDraftToPublishErrorResponse(
            fmt.Sprintf("forbidden: client %s is not driver of trip %s", req.ClientID, req.TripID),
        ))
	}

	if trip.Status == entity.StatusPublished {
        return c.Status(fiber.StatusNoContent).JSON(response.NewMoveTripDraftToPublishSuccessResponse(&trip))
	}

	if trip.Status != entity.StatusDraft {
        return c.Status(fiber.StatusConflict).JSON(response.NewMoveTripDraftToPublishErrorResponse(
            fmt.Sprintf("invalid trip status: expected %s, got %s", entity.StatusDraft, trip.Status),
        ))
	}

    oldStatus := trip.Status
	trip.Status = entity.StatusPublished

	updatedTrip, err := u.TripService.Update(c.Context(), &trip, oldStatus)
	if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(response.NewMoveTripDraftToPublishErrorResponse(
            "Failed to update trip",
            err.Error(),
        ))
	}

    return c.Status(fiber.StatusOK).JSON(response.NewMoveTripDraftToPublishSuccessResponse(updatedTrip))
}



