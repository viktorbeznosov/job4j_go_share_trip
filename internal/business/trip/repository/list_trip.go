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
	"job4j_go_share_trip/internal/business/trip/entity"
)

type TripListFilter struct {
	Viewer     access.Viewer
	ID         *uuid.UUID
	DriverID   *uuid.UUID
	ListParams access.ListParams
}

func (r *TripRepository) List(ctx context.Context, filter TripListFilter) ([]entity.Trip, int, error) {
	where, args := tripVisibilityAndFilters(filter)

	countQuery := "SELECT COUNT(*) FROM public.trips t WHERE " + where
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count trips: %w", err)
	}

	orderBy := tripOrderBy(filter.ListParams.SortBy, filter.ListParams.SortDir)
	limit := filter.ListParams.Limit
	offset := filter.ListParams.Offset
	args = append(args, limit, offset)

	listQuery := `
		SELECT
			t.id,
			t.driver_id,
			t.from_point,
			t.to_point,
			t.departure_time,
			t.seats,
			t.status,
			t.created_at
		FROM public.trips t
		WHERE ` + where + `
		ORDER BY ` + orderBy + `
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list trips: %w", err)
	}
	defer rows.Close()

	items := make([]entity.Trip, 0)
	for rows.Next() {
		var trip entity.Trip
		if err := rows.Scan(
			&trip.ID,
			&trip.DriverID,
			&trip.FromPoint,
			&trip.ToPoint,
			&trip.DepartureTime,
			&trip.Seats,
			&trip.Status,
			&trip.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, trip)
	}

	return items, total, rows.Err()
}

func (r *TripRepository) GetByIDForViewer(ctx context.Context, id uuid.UUID, viewer access.Viewer) (*entity.Trip, error) {
	where, args := tripVisibilityAndFilters(TripListFilter{
		Viewer: viewer,
		ID:     &id,
	})
	query := `
		SELECT
			t.id,
			t.driver_id,
			t.from_point,
			t.to_point,
			t.departure_time,
			t.seats,
			t.status,
			t.created_at
		FROM public.trips t
		WHERE ` + where

	var trip entity.Trip
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.DriverID,
		&trip.FromPoint,
		&trip.ToPoint,
		&trip.DepartureTime,
		&trip.Seats,
		&trip.Status,
		&trip.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tripErrors.ErrTripNotFound
		}
		return nil, fmt.Errorf("get trip for viewer: %w", err)
	}

	return &trip, nil
}

func tripVisibilityAndFilters(filter TripListFilter) (string, []any) {
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
		add("t.id = ?", *filter.ID)
	}

	if filter.Viewer.IsAdmin {
		if filter.DriverID != nil {
			add("t.driver_id = ?", *filter.DriverID)
		}
	} else {
		var vis []string
		if filter.Viewer.IsDriver {
			args = append(args, filter.Viewer.ClientID)
			vis = append(vis, fmt.Sprintf("t.driver_id = $%d", len(args)))
		}
		if filter.Viewer.IsPassenger {
			args = append(args, filter.Viewer.ClientID)
			vis = append(vis, fmt.Sprintf(
				"(t.status = 'published' OR EXISTS (SELECT 1 FROM public.trip_requests r WHERE r.trip_id = t.id AND r.passenger_id = $%d))",
				len(args),
			))
		}
		if filter.Viewer.IsDirector {
			vis = append(vis, fmt.Sprintf("t.status = '%s'", entity.StatusPreapproved))
		}
		if len(vis) == 0 {
			parts = append(parts, "FALSE")
		} else {
			parts = append(parts, "("+strings.Join(vis, " OR ")+")")
		}
	}

	p := filter.ListParams
	if p.DepartureFrom != nil {
		add("t.departure_time >= ?", *p.DepartureFrom)
	}
	if p.DepartureTo != nil {
		add("t.departure_time <= ?", *p.DepartureTo)
	}
	if p.FromPoint != "" {
		add("t.from_point = ?", p.FromPoint)
	}
	if p.ToPoint != "" {
		add("t.to_point = ?", p.ToPoint)
	}
	if p.Status != "" {
		add("t.status = ?", p.Status)
	}

	if len(parts) == 0 {
		return "TRUE", args
	}

	return strings.Join(parts, " AND "), args
}

func tripOrderBy(sortBy, sortDir string) string {
	column := "t.departure_time"
	if sortBy == access.SortByStatus {
		column = "t.status"
	}
	dir := "ASC"
	if sortDir == access.SortDirDesc {
		dir = "DESC"
	}
	return column + " " + dir
}
