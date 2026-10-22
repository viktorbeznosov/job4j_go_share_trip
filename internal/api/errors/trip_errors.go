package errors

import (
	"errors"
)

var (
	ErrFromPointRequired       = errors.New("from point is required")
	ErrToPointRequired         = errors.New("to point is required")
	ErrDepartureTimeRequired   = errors.New("departure time is required")
	ErrInvalidDateFormat       = errors.New("departure time must be in format '2006-01-02 15:04'")
	ErrDepartureTimePast       = errors.New("departure time cannot be in the past")
	ErrInvalidSeats            = errors.New("seats must be greater than 0")
	ErrSeatsTooHigh            = errors.New("seats cannot exceed 10")
	ErrTripIDRequired          = errors.New("trip id is required")
	ErrInvalidTripID           = errors.New("trip id must be a valid UUID")
	ErrCompanyIDRequired       = errors.New("company id is required")
	ErrInvalidCompanyID        = errors.New("company id must be a valid UUID")
	ErrDriverIDRequired        = errors.New("driver id is required")
	ErrRequestIDRequired       = errors.New("request id is required")
	ErrInvalidRequestID        = errors.New("request id must be a valid UUID")
	ErrConfirmRequired         = errors.New("confirm is required")
	ErrInvalidSortField        = errors.New("invalid sort field")
	ErrInvalidSortDir          = errors.New("invalid sort direction")
	ErrInvalidLimit            = errors.New("invalid limit")
	ErrInvalidOffset           = errors.New("invalid offset")
	ErrFilterNotAllowed        = errors.New("filter is not allowed")
	ErrPassengerIDRequired     = errors.New("passenger id is required")
	ErrInvalidPassengerID      = errors.New("passenger id must be a valid UUID")
	ErrTripPublishIsNotAllowed = errors.New("trip publish is not allowed")
	ErrTripStartIsNotAllowed   = errors.New("trip start is not allowed")
)

var (
	ErrTripNotFound    = errors.New("trip not found")
	ErrRequestNotFound = errors.New("request not found")
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInvalidStatus           = errors.New("invalid status")
	ErrUnknownStatus           = errors.New("unknown status")
	ErrTripAlreadyPublished    = errors.New("trip already published")
	ErrTripNotDraft            = errors.New("trip is not in draft status")
	ErrTripNotPreapproved      = errors.New("trip is not in preapproved status")
	ErrTripNotPublished        = errors.New("trip is not in published status")
	ErrTripNotStarted          = errors.New("trip is not in started status")
	ErrTripCancelNotAllowed    = errors.New("started trip cannot be canceled")
	ErrDriverNotOwner          = errors.New("driver is not the owner of the trip")
	ErrPassengerNotOwner       = errors.New("passenger is not the owner of the request")
	ErrRequestNotDraft         = errors.New("request is not in draft status")
	ErrRequestNotPublished     = errors.New("request is not in published status")
	ErrRequestNotPreAgreed     = errors.New("request is not in pre_agreed status")
	ErrRequestNotAgreed        = errors.New("request is not in agreed status")
	ErrRequestNotEnoughSeats   = errors.New("request seats exceed trip capacity")
	ErrConfirmRequiresDriver   = errors.New("only driver can confirm the request")
	ErrRequestConfirmForbidden = errors.New("client is not allowed to confirm or reject the request")
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
	case errors.Is(err, ErrCompanyIDRequired):
		return "Company ID is required"
	case errors.Is(err, ErrInvalidCompanyID):
		return "Company ID must be a valid UUID"
	case errors.Is(err, ErrDriverIDRequired):
		return "Driver ID is required"
	case errors.Is(err, ErrRequestIDRequired):
		return "Request ID is required"
	case errors.Is(err, ErrInvalidRequestID):
		return "Request ID must be a valid UUID"
	case errors.Is(err, ErrConfirmRequired):
		return "Confirm is required"
	case errors.Is(err, ErrInvalidSortField):
		return "Invalid sort field"
	case errors.Is(err, ErrInvalidSortDir):
		return "Invalid sort direction"
	case errors.Is(err, ErrInvalidLimit):
		return "Invalid limit"
	case errors.Is(err, ErrInvalidOffset):
		return "Invalid offset"
	case errors.Is(err, ErrFilterNotAllowed):
		return "Filter is not allowed"
	case errors.Is(err, ErrPassengerIDRequired):
		return "Passenger ID is required"
	case errors.Is(err, ErrInvalidPassengerID):
		return "Passenger ID must be a valid UUID"
	case errors.Is(err, ErrTripNotFound):
		return "Trip not found"
	case errors.Is(err, ErrRequestNotFound):
		return "Request not found"
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
	case errors.Is(err, ErrTripNotPreapproved):
		return "Trip is not in preapproved status"
	case errors.Is(err, ErrTripNotPublished):
		return "Trip is not in published status"
	case errors.Is(err, ErrTripNotStarted):
		return "Trip is not in started status"
	case errors.Is(err, ErrTripCancelNotAllowed):
		return "Started trip cannot be canceled"
	case errors.Is(err, ErrDriverNotOwner):
		return "Client is not driver of this trip"
	case errors.Is(err, ErrPassengerNotOwner):
		return "Client is not passenger of this request"
	case errors.Is(err, ErrRequestNotDraft):
		return "Request is not in draft status"
	case errors.Is(err, ErrRequestNotPublished):
		return "Request is not in published status"
	case errors.Is(err, ErrRequestNotPreAgreed):
		return "Request is not in pre_agreed status"
	case errors.Is(err, ErrRequestNotAgreed):
		return "Request is not in agreed status"
	case errors.Is(err, ErrRequestNotEnoughSeats):
		return "Request seats exceed trip capacity"
	case errors.Is(err, ErrConfirmRequiresDriver):
		return "Only driver can confirm the request"
	case errors.Is(err, ErrRequestConfirmForbidden):
		return "Client is not allowed to confirm or reject the request"
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
	case errors.Is(err, ErrTripNotFound),
		errors.Is(err, ErrRequestNotFound):
		return 404

	case errors.Is(err, ErrDriverNotOwner),
		errors.Is(err, ErrPassengerNotOwner),
		errors.Is(err, ErrConfirmRequiresDriver),
		errors.Is(err, ErrRequestConfirmForbidden),
		errors.Is(err, ErrTripPublishIsNotAllowed),
		errors.Is(err, ErrTripStartIsNotAllowed):
		return 403

	case errors.Is(err, ErrInvalidStatusTransition),
		errors.Is(err, ErrTripAlreadyPublished),
		errors.Is(err, ErrTripNotDraft),
		errors.Is(err, ErrTripNotPreapproved),
		errors.Is(err, ErrTripNotPublished),
		errors.Is(err, ErrTripNotStarted),
		errors.Is(err, ErrTripCancelNotAllowed),
		errors.Is(err, ErrRequestNotDraft),
		errors.Is(err, ErrRequestNotPublished),
		errors.Is(err, ErrRequestNotPreAgreed),
		errors.Is(err, ErrRequestNotAgreed),
		errors.Is(err, ErrRequestNotEnoughSeats):
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
		errors.Is(err, ErrCompanyIDRequired),
		errors.Is(err, ErrInvalidCompanyID),
		errors.Is(err, ErrDriverIDRequired),
		errors.Is(err, ErrRequestIDRequired),
		errors.Is(err, ErrInvalidRequestID),
		errors.Is(err, ErrConfirmRequired),
		errors.Is(err, ErrInvalidSortField),
		errors.Is(err, ErrInvalidSortDir),
		errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidOffset),
		errors.Is(err, ErrFilterNotAllowed),
		errors.Is(err, ErrPassengerIDRequired),
		errors.Is(err, ErrInvalidPassengerID),
		errors.Is(err, ErrUnknownStatus),
		errors.Is(err, ErrInvalidStatus):
		return 400

	default:
		return 500
	}
}
