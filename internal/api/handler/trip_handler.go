package handler

import "job4j_go_share_trip/internal/domain/trip/service"

type TripHandler struct {
	TripService *service.TripService
}

func NewTripHandler(tripService *service.TripService) *TripHandler {
	return &TripHandler{
		TripService: tripService,
	}
}

