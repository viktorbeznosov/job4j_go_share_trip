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
	Status  string                        `json:"status"`
	Message string                        `json:"message,omitempty"`
	Data    MoveTripDraftToPublishResponse `json:"data"`
	Error   string                        `json:"error,omitempty"`
}

type MoveTripFromPublishToStartedResponseWrapper struct {
	Status  string                             `json:"status"`
	Message string                             `json:"message,omitempty"`
	Data    MoveTripFromPublishToStartedResponse `json:"data"`
	Error   string                             `json:"error,omitempty"`
}

type ErrorResponseWrapper struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

