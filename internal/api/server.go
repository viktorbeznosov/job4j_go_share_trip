package api

import (
	"job4j_go_share_trip/internal/domain/trip/handler"
	"job4j_go_share_trip/internal/domain/trip/repository"
	trip_service "job4j_go_share_trip/internal/domain/trip/service"
	"job4j_go_share_trip/internal/shared/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct{
    TripHandler *handler.TripHandler
}

func NewServer(ppgxpool *pgxpool.Pool) *Server {
    tripService := trip_service.NewService(
        *repository.NewPostgresRepository(ppgxpool),
        *outbox.NewEventRepository(ppgxpool),
    )
	return &Server{
        TripHandler: handler.NewTripHandler(tripService),
	}
}
