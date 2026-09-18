package service

import (
	"context"

	"job4j_go_share_trip/internal/business/trip/domain"
	"job4j_go_share_trip/internal/business/trip/repository"
	"job4j_go_share_trip/internal/clients/contract"
	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"
)

type ContractChecker interface {
    CheckService(ctx context.Context, companyID, serviceCode string) (contract.CheckResult, error)
}

type TripService struct {
	tripDomain      domain.TripDomain
	tripRepository  repository.TripRepository
	eventRepository outbox.EventRepository
	contractClient  ContractClient   // ← интерфейс, не *contractclient.Client
	metrics         *metrics.Metrics
}

func NewService(
	tripDomain domain.TripDomain,
	tripRepository repository.TripRepository,
	eventRepository outbox.EventRepository,
	contractClient ContractClient,
	metrics *metrics.Metrics,
) *TripService {
	return &TripService{
		tripDomain:      tripDomain,
		tripRepository:  tripRepository,
		eventRepository: eventRepository,
		contractClient:  contractClient,
		metrics:         metrics,
	}
}