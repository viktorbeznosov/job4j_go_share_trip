package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) Route(route fiber.Router) {
	route.Get("/ready", s.Ready)
	route.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(s.Registry, promhttp.HandlerOpts{})))

	trip := route.Group("/trip")
	trip.Post("/", s.TripHandler.CreateTrip)
	trip.Put("/move_to_publish", s.TripHandler.MoveTripDraftToPublish)
	trip.Get("/:uuid", s.TripHandler.GetTripByUUID)


}
