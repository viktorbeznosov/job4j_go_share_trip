package service

import (
	"context"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/repository"
)

type TripService struct {
	repository repository.TripRepository
}

func NewService(repository repository.TripRepository) *TripService {
	return &TripService{
		repository: repository,
	}
}

func (s *TripService) Create(ctx context.Context, trip *entity.Trip) error {
	return s.repository.Create(ctx, trip)
}

func (s *TripService) Update(ctx context.Context, trip *entity.Trip, oldStatus entity.Status) (*entity.Trip, error) {
    return s.repository.Update(ctx, trip, oldStatus)
}

func (s *TripService) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	return s.repository.GetByTripID(ctx, tripID)
}

func (s *TripService) GetForUpdateByID(
    ctx context.Context,
    id uuid.UUID,
) (entity.Trip, error) {
    return s.repository.GetForUpdateByID(ctx, id)
}

