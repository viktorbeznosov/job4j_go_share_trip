package service

import (
	"job4j_go_share_trip/internal/domain/trip/repository"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"
)

type TripService struct {
	tripRepository repository.TripRepository
	eventRepository outbox.EventRepository
	metrics *metrics.Metrics
}

func NewService(
    tripRepository repository.TripRepository,
    eventRepository outbox.EventRepository,
    metrics *metrics.Metrics,
) *TripService {
	return &TripService{
		tripRepository: tripRepository,
		eventRepository: eventRepository,
		metrics: metrics,
	}
}