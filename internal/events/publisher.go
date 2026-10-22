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
	PublishTripCreated(ctx context.Context, event TripCreated) error
	PublishTripPreapproved(ctx context.Context, event TripPreapproved) error
	PublishTripPublished(ctx context.Context, event TripPublished) error
	PublishTripStarted(ctx context.Context, event TripStarted) error
	PublishTripCanceled(ctx context.Context, event TripCanceled) error
	PublishTripCompleted(ctx context.Context, event TripCompleted) error
	PublishTripPublishNotificated(ctx context.Context, event TripPublishNotificated) error
	PublishRequestEvent(ctx context.Context, event RequestEvent) error
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
	case outbox.EventTypeTripCreated:
		var tripCreated TripCreated
		if err := json.Unmarshal(event.Payload, &tripCreated); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		tripCreated.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(tripCreated.EventID, tripCreated.CorrelationID, tripCreated.CausationID, func() error {
			return p.kafka.PublishTripCreated(ctx, tripCreated)
		})
	case outbox.EventTypeTripPreapproved:
		var tripPreapproved TripPreapproved
		if err := json.Unmarshal(event.Payload, &tripPreapproved); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		tripPreapproved.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(tripPreapproved.EventID, tripPreapproved.CorrelationID, tripPreapproved.CausationID, func() error {
			return p.kafka.PublishTripPreapproved(ctx, tripPreapproved)
		})
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
	case outbox.EventTypeTripCanceled:
		var tripCanceled TripCanceled
		if err := json.Unmarshal(event.Payload, &tripCanceled); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		tripCanceled.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(tripCanceled.EventID, tripCanceled.CorrelationID, tripCanceled.CausationID, func() error {
			return p.kafka.PublishTripCanceled(ctx, tripCanceled)
		})
	case outbox.EventTypeTripCompleted:
		var tripCompleted TripCompleted
		if err := json.Unmarshal(event.Payload, &tripCompleted); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		tripCompleted.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(tripCompleted.EventID, tripCompleted.CorrelationID, tripCompleted.CausationID, func() error {
			return p.kafka.PublishTripCompleted(ctx, tripCompleted)
		})
	case outbox.EventTypeTripPublishNotificated:
		var notified TripPublishNotificated
		if err := json.Unmarshal(event.Payload, &notified); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		notified.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(notified.EventID, notified.CorrelationID, notified.CausationID, func() error {
			return p.kafka.PublishTripPublishNotificated(ctx, notified)
		})
	case outbox.EventTypeRequestCreated,
		outbox.EventTypeRequestPublished,
		outbox.EventTypeRequestPreconfirmed,
		outbox.EventTypeRequestConfirmed,
		outbox.EventTypeRequestRejected,
		outbox.EventTypeRequestStarted,
		outbox.EventTypeRequestCompleted,
		outbox.EventTypeRequestExpired:
		var requestEvent RequestEvent
		if err := json.Unmarshal(event.Payload, &requestEvent); err != nil {
			p.observePublish("error")
			p.metrics.OutboxPublishFailedTotal.Inc()
			_ = p.outbox.MarkFailed(ctx, event.ID, err)
			return nil
		}
		requestEvent.applyMetadata(metadataFromJSON(event.Metadata))
		return publish(requestEvent.EventID, requestEvent.CorrelationID, requestEvent.CausationID, func() error {
			return p.kafka.PublishRequestEvent(ctx, requestEvent)
		})
	default:
		return p.outbox.MarkSent(ctx, event.ID)
	}
}

func (p *Publisher) observePublish(result string) {
	p.metrics.OutboxPublishTotal.WithLabelValues(result).Inc()
}

func (e *TripCreated) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}

func (e *TripPreapproved) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
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

func (e *TripCanceled) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}

func (e *TripCompleted) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}

func (e *TripPublishNotificated) applyMetadata(meta EventMetadata) {
	merged := mergeEventMetadata(e.Metadata(), meta)
	e.EventID = merged.EventID
	e.EventType = merged.EventType
	e.CorrelationID = merged.CorrelationID
	e.CausationID = merged.CausationID
	e.Traceparent = merged.Traceparent
	e.OccurredAt = merged.OccurredAt
}
