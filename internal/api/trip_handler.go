package api

import (
	"job4j_go_share_trip/internal/business/trip/service"
)

type TripHandler struct {
	TripService  *service.TripService
	errorMapper  *ErrorMapper
}

func NewTripHandler(tripService *service.TripService) *TripHandler {
	return &TripHandler{
		TripService: tripService,
		errorMapper: NewErrorMapper(),
	}
}