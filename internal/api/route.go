package api

import (
	"job4j_go_share_trip/config"
	"job4j_go_share_trip/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) Route(route fiber.Router) {
	route.Get("/ready", s.Ready)
	route.Get("/metrics", adaptor.HTTPHandler(promhttp.HandlerFor(s.Registry, promhttp.HandlerOpts{})))

	trip := route.Group("/trip")
	trip.Post(
	    "/",
	    middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
	    s.TripHandler.CreateTrip,
	)
	trip.Put(
	    "/move_to_publish",
	    middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
	    s.TripHandler.MoveTripDraftToPublish,
	)
	trip.Get(
	    "/:uuid",
        middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
	    s.TripHandler.GetTripByUUID,
	)


}
