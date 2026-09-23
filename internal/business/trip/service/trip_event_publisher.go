package service

import (
	"context"

	"job4j_go_share_trip/internal/events"
)

type TripEventPublisher interface {
	PublishTripPublished(ctx context.Context, event events.TripPublished) error
}
