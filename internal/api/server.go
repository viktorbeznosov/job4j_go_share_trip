package api

import (
	"job4j_go_share_trip/internal/domain/trip/handler"
	"job4j_go_share_trip/internal/domain/trip/repository"
	trip_service "job4j_go_share_trip/internal/domain/trip/service"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/observability/metrics"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct{
    TripHandler *handler.TripHandler
    Registry    *prometheus.Registry
    Metrics     *metrics.Metrics
}

func NewServer(ppgxpool *pgxpool.Pool, registry *prometheus.Registry, m *metrics.Metrics) *Server {
    tripService := trip_service.NewService(
        *repository.NewPostgresRepository(ppgxpool, m),
        *outbox.NewEventRepository(ppgxpool, m),
        m,
    )
	return &Server{
        TripHandler: handler.NewTripHandler(tripService),
        Registry:    registry,
        Metrics:     m,
	}
}
