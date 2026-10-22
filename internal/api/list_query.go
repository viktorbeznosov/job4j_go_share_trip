package api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"job4j_go_share_trip/internal/access"
	tripErrors "job4j_go_share_trip/internal/api/errors"
	"job4j_go_share_trip/internal/middleware"
)

func viewerFromClaims(claims *middleware.KeycloakClaims) (access.Viewer, error) {
	clientID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return access.Viewer{}, err
	}

	return access.Viewer{
		ClientID:    clientID,
		IsAdmin:     claims.HasExactPermission(middleware.PermissionShareTripAdmin),
		IsDirector:  claims.HasExactPermission(middleware.PermissionShareTripDirector),
		IsDriver:    claims.HasExactPermission(middleware.PermissionShareTripDriver),
		IsPassenger: claims.HasExactPermission(middleware.PermissionShareTripPassenger),
	}, nil
}

func parseListParams(c *fiber.Ctx) (access.ListParams, error) {
	params := access.ListParams{
		FromPoint: c.Query("fromPoint"),
		ToPoint:   c.Query("toPoint"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sortBy"),
		SortDir:   c.Query("sortDir"),
	}

	from, err := parseOptionalQueryTime(c.Query("departureTimeFrom"))
	if err != nil {
		return access.ListParams{}, err
	}
	params.DepartureFrom = from

	to, err := parseOptionalQueryTime(c.Query("departureTimeTo"))
	if err != nil {
		return access.ListParams{}, err
	}
	params.DepartureTo = to

	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return access.ListParams{}, tripErrors.ErrInvalidLimit
		}
		params.Limit = limit
	}
	if raw := c.Query("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil {
			return access.ListParams{}, tripErrors.ErrInvalidOffset
		}
		params.Offset = offset
	}

	return params, nil
}

func parseOptionalQueryTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}

	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02 15:04", raw); err == nil {
		return &parsed, nil
	}

	return nil, tripErrors.ErrInvalidDateFormat
}

func parseOptionalUUIDQuery(raw string, invalid error) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, invalid
	}
	return &id, nil
}
