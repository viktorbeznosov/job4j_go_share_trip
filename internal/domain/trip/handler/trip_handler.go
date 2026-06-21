package handler

import "job4j_go_share_trip/internal/domain/trip/service"

type TripHandler struct {
	TripService *service.Service
}

func NewTripHandler(tripService *service.Service) *TripHandler {
	return &TripHandler{
		TripService: tripService,
	}
}

