package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/shared/outbox"
)

type MessagePublisher interface {
	PublishTripPublished(ctx context.Context, event TripPublished) error
}

type Publisher struct {
	outbox *outbox.EventRepository
	kafka  MessagePublisher
	logger *slog.Logger
}

func NewPublisher(
	outboxRepo *outbox.EventRepository,
	kafka MessagePublisher,
	logger *slog.Logger,
) *Publisher {
	return &Publisher{
		outbox: outboxRepo,
		kafka:  kafka,
		logger: logger,
	}
}

func (p *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.publishBatch(ctx); err != nil {
				p.logger.Error("publish outbox batch", "error", err)
			}
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context) error {
	return p.outbox.WithTx(ctx, func(ctx context.Context) error {
		events, err := p.outbox.LockPending(ctx, outbox.DefaultLockLimit)
		if err != nil {
			return err
		}

		for _, event := range events {
			var tripPublished TripPublished
			if err := json.Unmarshal(event.Payload, &tripPublished); err != nil {
				_ = p.outbox.MarkFailed(ctx, event.ID, err)
				continue
			}

			if err := p.kafka.PublishTripPublished(ctx, tripPublished); err != nil {
				_ = p.outbox.MarkFailed(ctx, event.ID, err)
				continue
			}

			if err := p.outbox.MarkSent(ctx, event.ID); err != nil {
				return err
			}
		}

		return nil
	})
}
