package request

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDriverIDRequired      = errors.New("driverId is required")
	ErrInvalidDriverID       = errors.New("driverId must be a valid UUID")
	ErrFromPointRequired     = errors.New("fromPoint is required")
	ErrToPointRequired       = errors.New("toPoint is required")
	ErrDepartureTimeRequired = errors.New("departureTime is required")
	ErrInvalidDateFormat     = errors.New("departureTime must be in format 'Y-m-d H:i' (e.g., 2026-06-25 10:00)")
	ErrDepartureTimePast     = errors.New("departureTime cannot be in the past")
	ErrInvalidSeats          = errors.New("seats must be greater than 0")
	ErrSeatsTooHigh          = errors.New("seats cannot exceed 10")
)

type CreateTripRequest struct {
	DriverID      uuid.UUID `json:"driverId"`
	FromPoint     string    `json:"fromPoint"`
	ToPoint       string    `json:"toPoint"`
	DepartureTime string    `json:"departureTime"` // Принимаем как строку
	Seats         int       `json:"seats"`
}

// Validate проверяет все поля запроса
func (r *CreateTripRequest) Validate() error {
	// 1. Проверка DriverID
	if r.DriverID == uuid.Nil {
		return ErrDriverIDRequired
	}
	if r.DriverID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidDriverID
	}

	// 2. Проверка FromPoint
	if r.FromPoint == "" {
		return ErrFromPointRequired
	}

	// 3. Проверка ToPoint
	if r.ToPoint == "" {
		return ErrToPointRequired
	}

	// 4. Проверка DepartureTime
	if r.DepartureTime == "" {
		return ErrDepartureTimeRequired
	}

	// 5. Проверка Seats
	if r.Seats <= 0 {
		return ErrInvalidSeats
	}
	if r.Seats > 10 {
		return ErrSeatsTooHigh
	}

	return nil
}

// ParseDepartureTime парсит строку в time.Time
func (r *CreateTripRequest) ParseDepartureTime() (time.Time, error) {
	// Формат: Y-m-d H:i
	layout := "2006-01-02 15:04"

	departureTime, err := time.Parse(layout, r.DepartureTime)
	if err != nil {
		return time.Time{}, ErrInvalidDateFormat
	}

	// Проверяем, что дата не в прошлом
	if departureTime.Before(time.Now()) {
		return time.Time{}, ErrDepartureTimePast
	}

	return departureTime, nil
}