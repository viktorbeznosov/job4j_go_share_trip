package response

import (
	"time"

	trip "job4j_go_share_trip/internal/domain/trip/entity"
)

type ItemResponse struct {
	ID            string            `json:"id"`
	DriverID      string            `json:"driverId"`
	FromPoint     string            `json:"fromPoint"`
	ToPoint       string            `json:"toPoint"`
	DepartureTime time.Time         `json:"departureTime"`
	Seats         int               `json:"seats"`
	Status        trip.Status     `json:"status"`
	CreatedAt     time.Time         `json:"createdAt"`
}

// FromEntity создает ItemResponse из entity.Trip
func (r *ItemResponse) FromEntity(trip *trip.Trip) {
	r.ID = trip.ID.String()
	r.DriverID = trip.DriverID.String()
	r.FromPoint = trip.FromPoint
	r.ToPoint = trip.ToPoint
	r.DepartureTime = trip.DepartureTime
	r.Seats = trip.Seats
	r.Status = trip.Status // Если есть поле Status
	r.CreatedAt = trip.CreatedAt
}

// NewItemResponse создает ItemResponse из entity.Trip
func NewItemResponse(trip *trip.Trip) ItemResponse {
	return ItemResponse{
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

// NewSuccessResponse создает успешный ответ
func NewSuccessResponse(trip *trip.Trip) map[string]interface{} {
	return map[string]interface{}{
		"status": "success",
		"data":   NewItemResponse(trip),
	}
}

// NewErrorResponse создает ответ с ошибкой
func NewErrorResponse(message string, details ...string) map[string]interface{} {
	resp := map[string]interface{}{
		"status":  "error",
		"message": message,
	}
	if len(details) > 0 {
		resp["details"] = details[0]
	}
	return resp
}