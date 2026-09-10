// internal/api/server.go
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

type Server struct {
	TripHandler *TripHandler
	Registry    *prometheus.Registry
	Metrics     *metrics.Metrics
}

func NewServer(
	pool *pgxpool.Pool,
	registry *prometheus.Registry,
	m *metrics.Metrics,
	contractClient trip_service.ContractClient,
) *Server {
	tripDomain := domain.NewDomain(
		*repository.NewPostgresRepository(pool, m),
		*outbox.NewEventRepository(pool, m),
		m,
	)

	tripService := trip_service.NewService(
		*tripDomain,
		*repository.NewPostgresRepository(pool, m),
		*outbox.NewEventRepository(pool, m),
		contractClient,   // ← прокидываем дальше
		m,
	)

	return &Server{
		TripHandler: NewTripHandler(tripService),
		Registry:    registry,
		Metrics:     m,
	}
}