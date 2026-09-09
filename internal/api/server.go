package api

import (
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/repository"
	trip_service "job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct{
    TripHandler *TripHandler
    Registry    *prometheus.Registry
    Metrics     *metrics.Metrics
}

func NewServer(ppgxpool *pgxpool.Pool, registry *prometheus.Registry, m *metrics.Metrics) *Server {
    tripDomain := domain.NewDomain(
        *repository.NewPostgresRepository(ppgxpool, m),
        *outbox.NewEventRepository(ppgxpool, m),
        m,
    )

    tripService := trip_service.NewService(
        *tripDomain,
        *repository.NewPostgresRepository(ppgxpool, m),
        *outbox.NewEventRepository(ppgxpool, m),
        m,
    )
	return &Server{
        TripHandler: NewTripHandler(tripService),
        Registry:    registry,
        Metrics:     m,
	}
}
