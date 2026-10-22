package domain

import (
	"job4j_go_share_trip/internal/access"
	tripErrors "job4j_go_share_trip/internal/api/errors"
)

func validateListParams(params access.ListParams, validStatus func(string) bool) error {
	if params.Status != "" && !validStatus(params.Status) {
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

func normalizeListParams(params access.ListParams) access.ListParams {
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
