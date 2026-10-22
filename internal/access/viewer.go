package access

import (
	"time"

	"github.com/google/uuid"
)

type Viewer struct {
	ClientID    uuid.UUID
	IsAdmin     bool
	IsDirector  bool
	IsDriver    bool
	IsPassenger bool
}

type ListParams struct {
	DepartureFrom *time.Time
	DepartureTo   *time.Time
	FromPoint     string
	ToPoint       string
	Status        string
	SortBy        string
	SortDir       string
	Limit         int
	Offset        int
}

const (
	SortByDepartureTime = "departureTime"
	SortByStatus        = "status"
	SortDirAsc          = "asc"
	SortDirDesc         = "desc"
	DefaultLimit        = 50
	MaxLimit            = 100
)
