package api

import "github.com/gofiber/fiber/v2"

func (s *Server) Route(route fiber.Router) {
	route.Get("/ready", s.Ready)

	trip := route.Group("/trip")
	trip.Post("/", s.TripHandler.CreateTrip)
	trip.Put("/move_to_publish", s.TripHandler.MoveTripDraftToPublish)
	trip.Get("/:uuid", s.TripHandler.GetTripByUUID)
}
