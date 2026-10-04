package contract

import (
	"context"
	"fmt"

	"github.com/go-resty/resty/v2"

	"job4j_go_share_trip/config"
	"job4j_go_share_trip/internal/business/trip/entity"
	"job4j_go_share_trip/internal/clients"
	"job4j_go_share_trip/internal/middleware"
	"job4j_go_share_trip/internal/observability/logctx"
)

type ClientInterface interface {
	CheckService(ctx context.Context, companyID string, serviceCode string) (CheckResult, error)
}

type Client struct {
	http *resty.Client
}

type CheckResult struct {
	Allowed bool
	Reason  string
}

type checkServiceResponse struct {
	Allowed bool   `json:"available"`
	Reason  string `json:"reason,omitempty"`
}

func NewClient() *Client {
	cfg := config.GetAppConfig()

	httpClient := clients.New(clients.Config{
		BaseURL:    cfg.ContractClient.BaseUrl,
		Timeout:    cfg.ContractClient.TimeOut,
		RetryCount: cfg.ContractClient.Retry,
	})

	return &Client{
		http: httpClient,
	}
}

func (c *Client) CheckService(ctx context.Context, companyID string, serviceCode entity.ServiceType) (CheckResult, error) {
	var response checkServiceResponse

	url := fmt.Sprintf("/api/companies/%s/services/%s/availability", companyID, serviceCode)

	req := c.http.R().
		SetContext(ctx).
		SetResult(&response)

	if requestID := logctx.RequestID(ctx); requestID != "" {
		req.SetHeader(middleware.RequestIDHeader, requestID)
	}
	if correlationID := logctx.CorrelationID(ctx); correlationID != "" {
		req.SetHeader(middleware.CorrelationIDHeader, correlationID)
	}
	if traceparent := logctx.Traceparent(ctx); traceparent != "" {
		req.SetHeader(middleware.TraceparentHeader, traceparent)
	}
	if tripID := logctx.TripID(ctx); tripID != "" {
		req.SetHeader(middleware.TripIDHeader, tripID)
	}
	if userID := logctx.UserID(ctx); userID != "" {
		req.SetHeader(middleware.UserIDHeader, userID)
	}

	resp, err := req.Get(url)

	if err != nil {
		return CheckResult{}, fmt.Errorf("failed to check service: %w", err)
	}

	if resp.IsError() {
		return CheckResult{}, fmt.Errorf("service check failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	return CheckResult(response), nil
}
