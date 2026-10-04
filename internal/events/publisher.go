package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/shared/outbox"
)

type MessagePublisher interface {
	PublishTripPublished(ctx context.Context, event TripPublished) error
	PublishTripStarted(ctx context.Context, event TripStarted) error
}

type Publisher struct {
	outbox  *outbox.EventRepository
	kafka   MessagePublisher
	logger  *slog.Logger
	metrics *metrics.Metrics
}

func NewPublisher(
	outboxRepo *outbox.EventRepository,
	kafka MessagePublisher,
	logger *slog.Logger,
	m *metrics.Metrics,
) *Publisher {
	return &Publisher{
		outbox:  outboxRepo,
		kafka:   kafka,
		logger:  logger,
		metrics: m,
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
	if pending, err := p.outbox.CountPending(ctx); err == nil {
		p.metrics.OutboxPendingTotal.Set(float64(pending))
	}

	return p.outbox.WithTx(ctx, func(ctx context.Context) error {
		events, err := p.outbox.LockPending(ctx, outbox.DefaultLockLimit)
		if err != nil {
			return err
		}

		for _, event := range events {
			if err := p.publishOne(ctx, event); err != nil {
				return err
			}
		}

		return nil
	})
}

func (p *Publisher) publishOne(ctx context.Context, event outbox.Event) error {
	publish := func(eventID, correlationID, causationID string, send func() error) error {
		if err := send(); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			p.logger.Error("outbox kafka publish failed",
				slog.String("operation", "OutboxPublish"),
				slog.String("event_id", eventID),
				slog.String("correlation_id", correlationID),
				slog.String("result", "error"),
				slog.Any("error", err),
			)
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}

		if err := p.outbox.MarkSent(ctx, event.ID); err != nil {
			return err
		}

		p.observePublish("success")
		p.logger.Info("outbox event published",
			slog.String("operation", "OutboxPublish"),
			slog.String("event_id", eventID),
			slog.String("correlation_id", correlationID),
			slog.String("causation_id", causationID),
			slog.String("event_type", string(event.EventType)),
			slog.String("result", "published"),
		)
		return nil
	}

	switch event.EventType {
	case outbox.EventTypeTripPublished:
		var tripPublished TripPublished
		if err := json.Unmarshal(event.Payload, &tripPublished); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		tripPublished.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(tripPublished.EventID, tripPublished.CorrelationID, tripPublished.CausationID, func() error {
			return p.kafka.PublishTripPublished(ctx, tripPublished)
		})
	case outbox.EventTypeTripStarted:
		var tripStarted TripStarted
		if err := json.Unmarshal(event.Payload, &tripStarted); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		tripStarted.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(tripStarted.EventID, tripStarted.CorrelationID, tripStarted.CausationID, func() error {
			return p.kafka.PublishTripStarted(ctx, tripStarted)
		})
	default:
		return p.outbox.MarkSent(ctx, event.ID)
	}
}

func (p *Publisher) observePublish(result string) {
	p.metrics.OutboxPublishTotal.WithLabelValues(result).Inc()
}

func (e *TripPublished) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}

func (e *TripStarted) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}
