package domain

import (
	requestRepo "job4j_go_share_trip/internal/business/request/repository"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"
)

type TripDomain struct {
	tripRepository    repository.TripRepository
	requestRepository requestRepo.RequestRepository
	eventRepository   outbox.EventRepository
	metrics           *metrics.Metrics
}

func NewDomain(
	tripRepository repository.TripRepository,
	requestRepository requestRepo.RequestRepository,
	eventRepository outbox.EventRepository,
	metrics *metrics.Metrics,
) *TripDomain {
	return &TripDomain{
		tripRepository:    tripRepository,
		requestRepository: requestRepository,
		eventRepository:   eventRepository,
		metrics:           metrics,
	}
}
