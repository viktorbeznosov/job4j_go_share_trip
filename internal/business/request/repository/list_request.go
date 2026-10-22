package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"job4j_go_share_trip/internal/access"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/business/request/entity"
)

type RequestListFilter struct {
	Viewer      access.Viewer
	ID          *uuid.UUID
	PassengerID *uuid.UUID
	ListParams  access.ListParams
}

func (r *RequestRepository) List(ctx context.Context, filter RequestListFilter) ([]entity.Request, int, error) {
	where, args := requestVisibilityAndFilters(filter)

	countQuery := "SELECT COUNT(*) FROM public.trip_requests req WHERE " + where
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count requests: %w", err)
	}

	orderBy := requestOrderBy(filter.ListParams.SortBy, filter.ListParams.SortDir)
	limit := filter.ListParams.Limit
	offset := filter.ListParams.Offset
	args = append(args, limit, offset)

	listQuery := selectRequestColumns + `
		FROM public.trip_requests req
		WHERE ` + where + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list requests: %w", err)
	}
	defer rows.Close()

	items := make([]entity.Request, 0)
	for rows.Next() {
		request, err := scanRequestRow(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *request)
	}

	return items, total, rows.Err()
}

func (r *RequestRepository) GetByIDForViewer(ctx context.Context, id uuid.UUID, viewer access.Viewer) (*entity.Request, error) {
	where, args := requestVisibilityAndFilters(RequestListFilter{
		Viewer: viewer,
		ID:     &id,
	})
	query := selectRequestColumns + `
		FROM public.trip_requests req
		WHERE ` + where

	request, err := scanRequestRow(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, tripErrors.ErrRequestNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get request for viewer: %w", err)
	}

	return request, nil
}

const selectRequestColumns = `
	SELECT
		req.id,
		req.passenger_id,
		req.from_point,
		req.to_point,
		req.departure_time,
		req.seats,
		req.status,
		req.trip_id,
		req.created_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequestRow(row rowScanner) (*entity.Request, error) {
	var request entity.Request
	err := row.Scan(
		&request.ID,
		&request.PassengerID,
		&request.FromPoint,
		&request.ToPoint,
		&request.DepartureTime,
		&request.Seats,
		&request.Status,
		&request.TripID,
		&request.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tripErrors.ErrRequestNotFound
		}
		return nil, fmt.Errorf("failed to scan request: %w", err)
	}

	return &request, nil
}

func requestVisibilityAndFilters(filter RequestListFilter) (string, []any) {
	var (
		parts []string
		args  []any
	)
	add := func(expr string, values ...any) {
		for _, value := range values {
			args = append(args, value)
			expr = strings.Replace(expr, "?", fmt.Sprintf("$%d", len(args)), 1)
		}
		parts = append(parts, expr)
	}

	if filter.ID != nil {
		add("req.id = ?", *filter.ID)
	}

	if filter.Viewer.IsAdmin {
		if filter.PassengerID != nil {
			add("req.passenger_id = ?", *filter.PassengerID)
		}
	} else {
		var vis []string
		if filter.Viewer.IsPassenger {
			args = append(args, filter.Viewer.ClientID)
			vis = append(vis, fmt.Sprintf("req.passenger_id = $%d", len(args)))
		}
		if filter.Viewer.IsDriver {
			args = append(args, filter.Viewer.ClientID)
			vis = append(vis, fmt.Sprintf(
				`(req.trip_id IS NOT NULL
				  AND req.status IN ('pre_agreed','agreed','started','canceled','completed')
				  AND req.trip_id IN (SELECT id FROM public.trips WHERE driver_id = $%d))`,
				len(args),
			))
		}
		if len(vis) == 0 {
			parts = append(parts, "FALSE")
		} else {
			parts = append(parts, "("+strings.Join(vis, " OR ")+")")
		}
	}

	p := filter.ListParams
	if p.DepartureFrom != nil {
		add("req.departure_time >= ?", *p.DepartureFrom)
	}
	if p.DepartureTo != nil {
		add("req.departure_time <= ?", *p.DepartureTo)
	}
	if p.FromPoint != "" {
		add("req.from_point = ?", p.FromPoint)
	}
	if p.ToPoint != "" {
		add("req.to_point = ?", p.ToPoint)
	}
	if p.Status != "" {
		add("req.status = ?", p.Status)
	}

	if len(parts) == 0 {
		return "TRUE", args
	}

	return strings.Join(parts, " AND "), args
}

func requestOrderBy(sortBy, sortDir string) string {
	column := "req.departure_time"
	if sortBy == access.SortByStatus {
		column = "req.status"
	}
	dir := "ASC"
	if sortDir == access.SortDirDesc {
		dir = "DESC"
	}
	return column + " " + dir
}
