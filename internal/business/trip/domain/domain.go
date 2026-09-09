package domain

import (
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/shared/outbox"
	"job4j_go_share_trip/internal/observability/metrics"
)

type TripDomain struct {
    tripRepository repository.TripRepository
    eventRepository outbox.EventRepository
    metrics *metrics.Metrics
}

func NewDomain(
    tripRepository repository.TripRepository,
    eventRepository outbox.EventRepository,
    metrics *metrics.Metrics,
) *TripDomain {
    return &TripDomain{
        tripRepository: tripRepository,
        eventRepository: eventRepository,
        metrics: metrics,
    }
}



