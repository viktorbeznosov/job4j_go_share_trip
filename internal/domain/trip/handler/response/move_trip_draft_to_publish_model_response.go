package response

import (
	trip "job4j_go_share_trip/internal/domain/trip/entity"
)

// MoveTripDraftToPublishResponse — ответ для публикации поездки
type MoveTripDraftToPublishResponse struct {
	TripID string `json:"tripId"`
}

// NewMoveTripDraftToPublishResponse создает ответ для публикации
func NewMoveTripDraftToPublishResponse(t *trip.Trip) MoveTripDraftToPublishResponse {
	return MoveTripDraftToPublishResponse{
		TripID: t.ID.String(),
	}
}