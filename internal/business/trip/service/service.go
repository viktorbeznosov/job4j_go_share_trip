package service

import (
	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"
)

type TripService struct {
    tripDomain domain.TripDomain
	tripRepository repository.TripRepository
	eventRepository outbox.EventRepository
	metrics *metrics.Metrics
}

func NewService(
    tripDomain domain.TripDomain,
    tripRepository repository.TripRepository,
    eventRepository outbox.EventRepository,
    metrics *metrics.Metrics,
) *TripService {
	return &TripService{
	    tripDomain: tripDomain,
		tripRepository: tripRepository,
		eventRepository: eventRepository,
		metrics: metrics,
	}
}