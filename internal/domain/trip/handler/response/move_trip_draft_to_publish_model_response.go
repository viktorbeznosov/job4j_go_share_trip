package response

import (
	trip "job4j_go_share_trip/internal/domain/trip/entity"
)

type MoveTripDraftToPublishModelResponse struct {
    TripID string
}

func NewMoveTripDraftToPublishModelResponse(trip *trip.Trip) MoveTripDraftToPublishModelResponse {
	return MoveTripDraftToPublishModelResponse{
		TripID: trip.ID.String(),
	}
}

func NewMoveTripDraftToPublishSuccessResponse(trip *trip.Trip) map[string]interface{} {
	return map[string]interface{}{
		"status": "success",
		"data":   NewMoveTripDraftToPublishModelResponse(trip),
	}
}

// NewErrorResponse создает ответ с ошибкой
func NewMoveTripDraftToPublishErrorResponse(message string, details ...string) map[string]interface{} {
	resp := map[string]interface{}{
		"status":  "error",
		"message": message,
	}
	if len(details) > 0 {
		resp["details"] = details[0]
	}
	return resp
}