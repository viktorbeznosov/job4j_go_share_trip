package domain

import (
	"context"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/access"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/business/trip/repository"
)

type ListTripsInput struct {
	Viewer     access.Viewer
	DriverID   *uuid.UUID
	ListParams access.ListParams
}

type ListTripsResult struct {
	Items  []GetTripResponse
	Total  int
	Limit  int
	Offset int
}

func (d *TripDomain) List(ctx context.Context, req ListTripsInput) (*ListTripsResult, error) {
	if err := validateTripListInput(req); err != nil {
		return nil, err
	}

	params := normalizeListParams(req.ListParams)
	items, total, err := d.tripRepository.List(ctx, repository.TripListFilter{
		Viewer:     req.Viewer,
		DriverID:   req.DriverID,
		ListParams: params,
	})
	if err != nil {
		return nil, err
	}

	result := make([]GetTripResponse, 0, len(items))
	for i := range items {
		result = append(result, mapTrip(&items[i]))
	}

	return &ListTripsResult{
		Items:  result,
		Total:  total,
		Limit:  params.Limit,
		Offset: params.Offset,
	}, nil
}

func (d *TripDomain) GetByTripIDForViewer(ctx context.Context, tripID uuid.UUID, viewer access.Viewer) (*GetTripResponse, error) {
	trip, err := d.tripRepository.GetByIDForViewer(ctx, tripID, viewer)
	if err != nil {
		return nil, err
	}

	resp := mapTrip(trip)
	return &resp, nil
}

func mapTrip(trip *entity.Trip) GetTripResponse {
	return GetTripResponse{
		ID:            trip.ID,
		DriverID:      trip.DriverID,
		FromPoint:     trip.FromPoint,
		ToPoint:       trip.ToPoint,
		DepartureTime: trip.DepartureTime,
		Seats:         trip.Seats,
		Status:        string(trip.Status),
		CreatedAt:     trip.CreatedAt,
	}
}

func validateTripListInput(req ListTripsInput) error {
	if req.DriverID != nil && !req.Viewer.IsAdmin {
		return tripErrors.ErrFilterNotAllowed
	}
	if err := validateListParams(req.ListParams, validTripStatus); err != nil {
		return err
	}
	return nil
}

func validTripStatus(status string) bool {
	switch entity.Status(status) {
	case entity.StatusDraft, entity.StatusPreapproved, entity.StatusPublished,
		entity.StatusStarted, entity.StatusCanceled, entity.StatusConfirmed:
		return true
	default:
		return false
	}
}
