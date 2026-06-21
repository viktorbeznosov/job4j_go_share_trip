package service

import (
	"context"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/domain/trip/entity"
	"job4j_go_share_trip/internal/domain/trip/repository"
)

type Service struct {
	repository repository.Repository
}

func NewService(repository repository.Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, trip *entity.Trip) error {
	return s.repository.Create(ctx, trip)
}

func (s *Service) GetByTripID(ctx context.Context, tripID uuid.UUID) (*entity.Trip, error) {
	return s.repository.GetByTripID(ctx, tripID)
}


