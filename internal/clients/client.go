package clients

import (
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
)

type Config struct {
	BaseURL         string
	Timeout         int
	RetryCount      int
}

func New(cfg Config) *resty.Client {
	client := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetTimeout(time.Duration(cfg.Timeout) * time.Second).
		SetRetryCount(cfg.RetryCount).
		SetRetryWaitTime(200 * time.Millisecond).
		SetRetryMaxWaitTime(1 * time.Second)

	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return true
		}
		return r.StatusCode() == http.StatusTooManyRequests ||
			r.StatusCode() == http.StatusBadGateway ||
			r.StatusCode() == http.StatusServiceUnavailable ||
			r.StatusCode() == http.StatusGatewayTimeout
	})

	return client
}