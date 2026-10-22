package api

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type CreateTripResponseWrapper struct {
	Status  string             `json:"status"`
	Message string             `json:"message,omitempty"`
	Data    CreateTripResponse `json:"data"`
	Error   string             `json:"error,omitempty"`
}

type MoveTripDraftToPublishResponseWrapper struct {
	Status  string                         `json:"status"`
	Message string                         `json:"message,omitempty"`
	Data    MoveTripDraftToPublishResponse `json:"data"`
	Error   string                         `json:"error,omitempty"`
}

type ApproveTripResponseWrapper struct {
	Status  string              `json:"status"`
	Message string              `json:"message,omitempty"`
	Data    ApproveTripResponse `json:"data"`
	Error   string              `json:"error,omitempty"`
}

type MoveTripFromPublishToStartedResponseWrapper struct {
	Status  string                               `json:"status"`
	Message string                               `json:"message,omitempty"`
	Data    MoveTripFromPublishToStartedResponse `json:"data"`
	Error   string                               `json:"error,omitempty"`
}

type MoveTripToCanceledResponseWrapper struct {
	Status  string                     `json:"status"`
	Message string                     `json:"message,omitempty"`
	Data    MoveTripToCanceledResponse `json:"data"`
	Error   string                     `json:"error,omitempty"`
}

type MoveTripToCompletedResponseWrapper struct {
	Status  string                      `json:"status"`
	Message string                      `json:"message,omitempty"`
	Data    MoveTripToCompletedResponse `json:"data"`
	Error   string                      `json:"error,omitempty"`
}

type CreateRequestResponseWrapper struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    RequestResponse `json:"data"`
	Error   string          `json:"error,omitempty"`
}

type MoveRequestToPublishResponseWrapper struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    RequestResponse `json:"data"`
	Error   string          `json:"error,omitempty"`
}

type MoveRequestToAgreedResponseWrapper struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    RequestResponse `json:"data"`
	Error   string          `json:"error,omitempty"`
}

type ConfirmRequestResponseWrapper struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    RequestResponse `json:"data"`
	Error   string          `json:"error,omitempty"`
}

type ListTripsResponseWrapper struct {
	Status  string            `json:"status"`
	Message string            `json:"message,omitempty"`
	Data    ListTripsResponse `json:"data"`
	Error   string            `json:"error,omitempty"`
}

type ListRequestsResponseWrapper struct {
	Status  string               `json:"status"`
	Message string               `json:"message,omitempty"`
	Data    ListRequestsResponse `json:"data"`
	Error   string               `json:"error,omitempty"`
}

type GetTripResponseWrapper struct {
	Status  string             `json:"status"`
	Message string             `json:"message,omitempty"`
	Data    CreateTripResponse `json:"data"`
	Error   string             `json:"error,omitempty"`
}

type GetRequestResponseWrapper struct {
	Status  string          `json:"status"`
	Message string          `json:"message,omitempty"`
	Data    RequestResponse `json:"data"`
	Error   string          `json:"error,omitempty"`
}

type ErrorResponseWrapper struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
