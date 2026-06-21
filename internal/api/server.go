package api

import (
	"job4j_go_share_trip/internal/domain/trip/handler"
	"job4j_go_share_trip/internal/domain/trip/repository"
	trip_service "job4j_go_share_trip/internal/domain/trip/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct{
    TripHandler *handler.TripHandler
}

func NewServer(ppgxpool *pgxpool.Pool) *Server {
    tripService := trip_service.NewService(*repository.NewPostgresRepository(ppgxpool))
	return &Server{
        TripHandler: handler.NewTripHandler(tripService),
	}
}
