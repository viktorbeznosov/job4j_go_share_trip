package domain

import (
	"context"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/access"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/entity"
	"job4j_go_share_trip/internal/business/request/repository"
)

type ListRequestsInput struct {
	Viewer      access.Viewer
	PassengerID *uuid.UUID
	ListParams  access.ListParams
}

type ListRequestsResult struct {
	Items  []RequestResponse
	Total  int
	Limit  int
	Offset int
}

func (d *RequestDomain) List(ctx context.Context, req ListRequestsInput) (*ListRequestsResult, error) {
	if req.PassengerID != nil && !req.Viewer.IsAdmin {
		return nil, tripErrors.ErrFilterNotAllowed
	}
	if err := validateRequestListParams(req.ListParams); err != nil {
		return nil, err
	}

	params := normalizeRequestListParams(req.ListParams)
	items, total, err := d.requestRepository.List(ctx, repository.RequestListFilter{
		Viewer:      req.Viewer,
		PassengerID: req.PassengerID,
		ListParams:  params,
	})
	if err != nil {
		return nil, err
	}

	result := make([]RequestResponse, 0, len(items))
	for i := range items {
		result = append(result, *newRequestResponse(&items[i]))
	}

	return &ListRequestsResult{
		Items:  result,
		Total:  total,
		Limit:  params.Limit,
		Offset: params.Offset,
	}, nil
}

func (d *RequestDomain) GetByIDForViewer(ctx context.Context, requestID uuid.UUID, viewer access.Viewer) (*RequestResponse, error) {
	request, err := d.requestRepository.GetByIDForViewer(ctx, requestID, viewer)
	if err != nil {
		return nil, err
	}

	return newRequestResponse(request), nil
}

func validateRequestListParams(params access.ListParams) error {
	if params.Status != "" && !validRequestStatus(params.Status) {
		return tripErrors.ErrUnknownStatus
	}
	if params.SortBy != "" && params.SortBy != access.SortByDepartureTime && params.SortBy != access.SortByStatus {
		return tripErrors.ErrInvalidSortField
	}
	if params.SortDir != "" && params.SortDir != access.SortDirAsc && params.SortDir != access.SortDirDesc {
		return tripErrors.ErrInvalidSortDir
	}
	if params.Limit < 0 || params.Limit > access.MaxLimit {
		return tripErrors.ErrInvalidLimit
	}
	if params.Offset < 0 {
		return tripErrors.ErrInvalidOffset
	}
	return nil
}

func normalizeRequestListParams(params access.ListParams) access.ListParams {
	if params.Limit == 0 {
		params.Limit = access.DefaultLimit
	}
	if params.SortBy == "" {
		params.SortBy = access.SortByDepartureTime
	}
	if params.SortDir == "" {
		params.SortDir = access.SortDirAsc
	}
	return params
}

func validRequestStatus(status string) bool {
	switch entity.Status(status) {
	case entity.StatusDraft, entity.StatusPublished, entity.StatusPreAgreed,
		entity.StatusAgreed, entity.StatusStarted, entity.StatusCanceled, entity.StatusCompleted:
		return true
	default:
		return false
	}
}
