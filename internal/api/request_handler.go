package api

import (
	"time"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/business/request/service"
)

type RequestHandler struct {
	RequestService *service.RequestService
	errorMapper    *ErrorMapper
}

func NewRequestHandler(requestService *service.RequestService) *RequestHandler {
	return &RequestHandler{
		RequestService: requestService,
		errorMapper:    NewErrorMapper(),
	}
}

type RequestResponse struct {
	ID            string    `json:"id"`
	PassengerID   string    `json:"passengerId"`
	FromPoint     string    `json:"fromPoint"`
	ToPoint       string    `json:"toPoint"`
	DepartureTime time.Time `json:"departureTime"`
	Seats         int       `json:"seats"`
	Status        string    `json:"status"`
	TripID        *string   `json:"tripId"`
	CreatedAt     time.Time `json:"createdAt"`
}

func newRequestResponse(resp service.RequestResponse) RequestResponse {
	var tripID *string
	if resp.TripID != nil {
		value := resp.TripID.String()
		tripID = &value
	}

	return RequestResponse{
		ID:            resp.ID.String(),
		PassengerID:   resp.PassengerID.String(),
		FromPoint:     resp.FromPoint,
		ToPoint:       resp.ToPoint,
		DepartureTime: resp.DepartureTime,
		Seats:         resp.Seats,
		Status:        resp.Status,
		TripID:        tripID,
		CreatedAt:     resp.CreatedAt,
	}
}

func parseClientID(subject string) (uuid.UUID, error) {
	return uuid.Parse(subject)
}
