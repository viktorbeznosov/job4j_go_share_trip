package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"job4j_go_share_trip/internal/access"
	"job4j_go_share_trip/internal/business/request/domain"
	"job4j_go_share_trip/internal/observability/logctx"
)

type RequestService struct {
	requestDomain domain.RequestDomain
}

func NewService(requestDomain domain.RequestDomain) *RequestService {
	return &RequestService{
		requestDomain: requestDomain,
	}
}

type RequestResponse struct {
	ID            uuid.UUID
	PassengerID   uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
	Status        string
	TripID        *uuid.UUID
	CreatedAt     time.Time
}

func mapResponse(resp *domain.RequestResponse) *RequestResponse {
	return &RequestResponse{
		ID:            resp.ID,
		PassengerID:   resp.PassengerID,
		FromPoint:     resp.FromPoint,
		ToPoint:       resp.ToPoint,
		DepartureTime: resp.DepartureTime,
		Seats:         resp.Seats,
		Status:        resp.Status,
		TripID:        resp.TripID,
		CreatedAt:     resp.CreatedAt,
	}
}

type CreateRequestInput struct {
	PassengerID   uuid.UUID
	FromPoint     string
	ToPoint       string
	DepartureTime time.Time
	Seats         int
}

func (s *RequestService) Create(ctx context.Context, req CreateRequestInput) (*RequestResponse, error) {
	logger := logctx.Logger(ctx).With(
		slog.String("service", "RequestService"),
		slog.String("operation", "CreateRequest"),
		slog.String("client_id", req.PassengerID.String()),
	)
	logger.Info("create request started")

	resp, err := s.requestDomain.Create(ctx, domain.CreateRequestInput{
		PassengerID:   req.PassengerID,
		FromPoint:     req.FromPoint,
		ToPoint:       req.ToPoint,
		DepartureTime: req.DepartureTime,
		Seats:         req.Seats,
	})
	if err != nil {
		logger.Error("failed to create request", slog.Any("error", err))
		return nil, err
	}

	logger.Info("create request completed", slog.String("request_id", resp.ID.String()))
	return mapResponse(resp), nil
}

func (s *RequestService) GetByID(ctx context.Context, requestID uuid.UUID) (*RequestResponse, error) {
	resp, err := s.requestDomain.GetByID(ctx, domain.GetRequestInput{RequestID: requestID})
	if err != nil {
		return nil, err
	}
	return mapResponse(resp), nil
}

func (s *RequestService) GetByIDForViewer(ctx context.Context, requestID uuid.UUID, viewer access.Viewer) (*RequestResponse, error) {
	resp, err := s.requestDomain.GetByIDForViewer(ctx, requestID, viewer)
	if err != nil {
		return nil, err
	}
	return mapResponse(resp), nil
}

type ListRequestsInput struct {
	Viewer      access.Viewer
	PassengerID *uuid.UUID
	ListParams  access.ListParams
}

type ListRequestsResult struct {
	Items  []RequestResponse
	Total  int
	Limit  int
	Offset int
}

func (s *RequestService) List(ctx context.Context, req ListRequestsInput) (*ListRequestsResult, error) {
	resp, err := s.requestDomain.List(ctx, domain.ListRequestsInput{
		Viewer:      req.Viewer,
		PassengerID: req.PassengerID,
		ListParams:  req.ListParams,
	})
	if err != nil {
		return nil, err
	}

	items := make([]RequestResponse, 0, len(resp.Items))
	for i := range resp.Items {
		items = append(items, *mapResponse(&resp.Items[i]))
	}

	return &ListRequestsResult{
		Items:  items,
		Total:  resp.Total,
		Limit:  resp.Limit,
		Offset: resp.Offset,
	}, nil
}

type MoveToPublishResult struct {
	Request          *RequestResponse
	AlreadyPublished bool
}

func (s *RequestService) MoveToPublish(ctx context.Context, requestID, passengerID uuid.UUID) (*MoveToPublishResult, error) {
	resp, err := s.requestDomain.MoveToPublish(ctx, domain.MoveToPublishInput{
		RequestID:   requestID,
		PassengerID: passengerID,
	})
	if err != nil {
		return nil, err
	}
	return &MoveToPublishResult{
		Request:          mapResponse(resp.Request),
		AlreadyPublished: resp.AlreadyPublished,
	}, nil
}

func (s *RequestService) MoveToAgreed(ctx context.Context, requestID, passengerID, tripID uuid.UUID) (*RequestResponse, error) {
	resp, err := s.requestDomain.MoveToAgreed(ctx, domain.MoveToAgreedInput{
		RequestID:   requestID,
		PassengerID: passengerID,
		TripID:      tripID,
	})
	if err != nil {
		return nil, err
	}
	return mapResponse(resp), nil
}

type ConfirmResult struct {
	Request       *RequestResponse
	AlreadyAgreed bool
}

func (s *RequestService) Confirm(ctx context.Context, requestID, clientID uuid.UUID, confirm bool) (*ConfirmResult, error) {
	resp, err := s.requestDomain.Confirm(ctx, domain.ConfirmRequestInput{
		RequestID: requestID,
		ClientID:  clientID,
		Confirm:   confirm,
	})
	if err != nil {
		return nil, err
	}
	return &ConfirmResult{
		Request:       mapResponse(resp.Request),
		AlreadyAgreed: resp.AlreadyAgreed,
	}, nil
}
