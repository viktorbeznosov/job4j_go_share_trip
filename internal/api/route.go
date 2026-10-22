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
		middleware.RequireUserPermission(middleware.PermissionShareTripDriver),
		s.TripHandler.CreateTrip,
	)
	trip.Put(
		"/move_to_publish",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripDriver),
		s.TripHandler.MoveTripDraftToPublish,
	)
	trip.Put(
		"/move_to_started",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripDriver),
		s.TripHandler.MoveTripFromPublishToStarted,
	)
	trip.Put(
		"/move_to_canceled",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripDriver),
		s.TripHandler.MoveTripToCanceled,
	)
	trip.Put(
		"/move_to_completed",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripDriver),
		s.TripHandler.MoveTripToCompleted,
	)
	trip.Put(
		"/approve",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripDirector),
		s.TripHandler.ApproveTrip,
	)
	trip.Get(
		"/",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireAnyUserPermission(
			middleware.PermissionShareTripDriver,
			middleware.PermissionShareTripPassenger,
			middleware.PermissionShareTripDirector,
			middleware.PermissionShareTripAdmin,
		),
		s.TripHandler.ListTrips,
	)
	trip.Get(
		"/:uuid",
		middleware.RequireClientRole(config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api"), "client"),
		middleware.RequireAnyUserPermission(
			middleware.PermissionShareTripDriver,
			middleware.PermissionShareTripPassenger,
			middleware.PermissionShareTripDirector,
			middleware.PermissionShareTripAdmin,
		),
		s.TripHandler.GetTripByUUID,
	)

	clientID := config.Env("KEYCLOAK_CLIENT_ID", "sharetrip-api")
	request := route.Group("/request")
	request.Post(
		"/",
		middleware.RequireClientRole(clientID, "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripPassenger),
		s.RequestHandler.CreateRequest,
	)
	request.Put(
		"/move_to_publish",
		middleware.RequireClientRole(clientID, "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripPassenger),
		s.RequestHandler.MoveRequestToPublish,
	)
	request.Put(
		"/move_request_to_agreed",
		middleware.RequireClientRole(clientID, "client"),
		middleware.RequireUserPermission(middleware.PermissionShareTripPassenger),
		s.RequestHandler.MoveRequestToAgreed,
	)
	request.Put(
		"/confirm",
		middleware.RequireClientRole(clientID, "client"),
		middleware.RequireAnyUserPermission(
			middleware.PermissionShareTripDriver,
			middleware.PermissionShareTripPassenger,
		),
		s.RequestHandler.ConfirmRequest,
	)
	request.Get(
		"/",
		middleware.RequireClientRole(clientID, "client"),
		middleware.RequireAnyUserPermission(
			middleware.PermissionShareTripPassenger,
			middleware.PermissionShareTripDriver,
			middleware.PermissionShareTripAdmin,
		),
		s.RequestHandler.ListRequests,
	)
	request.Get(
		"/:uuid",
		middleware.RequireClientRole(clientID, "client"),
		middleware.RequireAnyUserPermission(
			middleware.PermissionShareTripPassenger,
			middleware.PermissionShareTripDriver,
			middleware.PermissionShareTripAdmin,
		),
		s.RequestHandler.GetRequestByUUID,
	)
}
