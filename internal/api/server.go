// internal/api/server.go
package api

import (
	requestDomain "job4j_go_share_trip/internal/business/request/domain"
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	requestService "job4j_go_share_trip/internal/business/request/service"
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/repository"
	trip_service "job4j_go_share_trip/internal/business/trip/service"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct {
	TripHandler    *TripHandler
	RequestHandler *RequestHandler
	Registry       *prometheus.Registry
	Metrics        *metrics.Metrics
}

func NewServer(
	pool *pgxpool.Pool,
	registry *prometheus.Registry,
	m *metrics.Metrics,
	contractClient trip_service.ContractClient,
	tripPublisher trip_service.TripEventPublisher,
) *Server {
	tripDomain := domain.NewDomain(
		*repository.NewPostgresRepository(pool, m),
		*requestRepo.NewPostgresRepository(pool, m),
		*outbox.NewEventRepository(pool, m),
		m,
	)

	tripService := trip_service.NewService(
		*tripDomain,
		*repository.NewPostgresRepository(pool, m),
		*outbox.NewEventRepository(pool, m),
		contractClient,
		tripPublisher,
		m,
	)

	tripRepository := repository.NewPostgresRepository(pool, m)
	reqDomain := requestDomain.NewDomain(
		*requestRepo.NewPostgresRepository(pool, m),
		*tripRepository,
		*outbox.NewEventRepository(pool, m),
	)

	return &Server{
		TripHandler:    NewTripHandler(tripService),
		RequestHandler: NewRequestHandler(requestService.NewService(*reqDomain)),
		Registry:       registry,
		Metrics:        m,
	}
}
