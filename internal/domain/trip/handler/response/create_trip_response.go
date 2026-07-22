package response

import (
	"time"

	trip "job4j_go_share_trip/internal/domain/trip/entity"
)

// ItemResponse — структура для одного элемента
type ItemResponse struct {
	ID            string    `json:"id"`
	DriverID      string    `json:"driverId"`
	FromPoint     string    `json:"fromPoint"`
	ToPoint       string    `json:"toPoint"`
	DepartureTime time.Time `json:"departureTime"`
	Seats         int       `json:"seats"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

// NewItemResponse создает ItemResponse из entity.Trip
func NewItemResponse(t *trip.Trip) ItemResponse {
	return ItemResponse{
		ID:            t.ID.String(),
		DriverID:      t.DriverID.String(),
		FromPoint:     t.FromPoint,
		ToPoint:       t.ToPoint,
		DepartureTime: t.DepartureTime,
		Seats:         t.Seats,
		Status:        string(t.Status),
		CreatedAt:     t.CreatedAt,
	}
}

// SuccessResponse — общий успешный ответ
type SuccessResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// ErrorResponse — общий ответ с ошибкой
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewSuccessResponse создает успешный ответ
func NewSuccessResponse(data interface{}) SuccessResponse {
	return SuccessResponse{
		Status: "success",
		Data:   data,
	}
}

// NewErrorResponse создает ответ с ошибкой
func NewErrorResponse(message string, details ...string) ErrorResponse {
	resp := ErrorResponse{
		Status:  "error",
		Message: message,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	return resp
}