package domain

import (
	"context"

	"github.com/google/uuid"
)

type GetRequestInput struct {
	RequestID uuid.UUID
}

func (d *RequestDomain) GetByID(ctx context.Context, req GetRequestInput) (*RequestResponse, error) {
	request, err := d.requestRepository.GetByID(ctx, req.RequestID)
	if err != nil {
		return nil, err
	}

	return newRequestResponse(request), nil
}
