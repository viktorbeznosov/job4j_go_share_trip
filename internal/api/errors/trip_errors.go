package errors

import (
	"errors"
)

var (
	ErrFromPointRequired = errors.New("from point is required")
	ErrToPointRequired = errors.New("to point is required")
	ErrDepartureTimeRequired = errors.New("departure time is required")
	ErrInvalidDateFormat = errors.New("departure time must be in format '2006-01-02 15:04'")
	ErrDepartureTimePast = errors.New("departure time cannot be in the past")
	ErrInvalidSeats = errors.New("seats must be greater than 0")
	ErrSeatsTooHigh = errors.New("seats cannot exceed 10")
	ErrTripIDRequired = errors.New("trip id is required")
	ErrInvalidTripID = errors.New("trip id must be a valid UUID")
	ErrDriverIDRequired = errors.New("driver id is required")
	ErrTripPublishIsNotAllowed = errors.New("trip publish is not allowed")
	ErrTripStartIsNotAllowed = errors.New("trip start is not allowed")
)

var (
	ErrTripNotFound = errors.New("trip not found")
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidStatus = errors.New("invalid status")
	ErrUnknownStatus = errors.New("unknown status")
	ErrTripAlreadyPublished = errors.New("trip already published")
	ErrTripNotDraft = errors.New("trip is not in draft status")
	ErrTripNotPublished = errors.New("trip is not in published status")
	ErrDriverNotOwner = errors.New("driver is not the owner of the trip")
)

var (
	ErrServiceFailed = errors.New("service operation failed")
)

func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrFromPointRequired):
		return "From point is required"
	case errors.Is(err, ErrToPointRequired):
		return "To point is required"
	case errors.Is(err, ErrDepartureTimeRequired):
		return "Departure time is required"
	case errors.Is(err, ErrInvalidDateFormat):
		return "Invalid date format. Use '2006-01-02 15:04'"
	case errors.Is(err, ErrDepartureTimePast):
		return "Departure time cannot be in the past"
	case errors.Is(err, ErrInvalidSeats):
		return "Seats must be greater than 0"
	case errors.Is(err, ErrSeatsTooHigh):
		return "Seats cannot exceed 10"
	case errors.Is(err, ErrTripIDRequired):
		return "Trip ID is required"
	case errors.Is(err, ErrInvalidTripID):
		return "Trip ID must be a valid UUID"
	case errors.Is(err, ErrDriverIDRequired):
		return "Driver ID is required"
	case errors.Is(err, ErrTripNotFound):
		return "Trip not found"
	case errors.Is(err, ErrInvalidStatusTransition):
		return "Invalid status transition"
	case errors.Is(err, ErrInvalidStatus):
		return "Invalid status"
	case errors.Is(err, ErrUnknownStatus):
		return "Unknown status"
	case errors.Is(err, ErrTripAlreadyPublished):
		return "Trip already published"
	case errors.Is(err, ErrTripNotDraft):
		return "Trip is not in draft status"
	case errors.Is(err, ErrTripNotPublished):
		return "Trip is not in published status"
	case errors.Is(err, ErrDriverNotOwner):
		return "Driver is not the owner of the trip"
	case errors.Is(err, ErrTripPublishIsNotAllowed):
		return "Publish trips is not allowed for this company"
	case errors.Is(err, ErrTripStartIsNotAllowed):
		return "Start trips is not allowed for this company"

	default:
		return "Internal server error"
	}
}

func GetHTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrTripNotFound):
		return 404

	case errors.Is(err, ErrDriverNotOwner),
		errors.Is(err, ErrTripPublishIsNotAllowed),
		errors.Is(err, ErrTripStartIsNotAllowed):
		return 403

	case errors.Is(err, ErrInvalidStatusTransition),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrUnknownStatus),
		errors.Is(err, ErrTripAlreadyPublished),
		errors.Is(err, ErrTripNotDraft),
		errors.Is(err, ErrTripNotPublished):
		return 409

	case errors.Is(err, ErrFromPointRequired),
		errors.Is(err, ErrToPointRequired),
		errors.Is(err, ErrDepartureTimeRequired),
		errors.Is(err, ErrInvalidDateFormat),
		errors.Is(err, ErrDepartureTimePast),
		errors.Is(err, ErrInvalidSeats),
		errors.Is(err, ErrSeatsTooHigh),
		errors.Is(err, ErrTripIDRequired),
		errors.Is(err, ErrInvalidTripID),
		errors.Is(err, ErrDriverIDRequired):
		return 400

	default:
		return 500
	}
}