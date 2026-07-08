package outbox

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j_go_share_trip/internal/observability/metrics"
)

type EventRepository struct {
	db *pgxpool.Pool
	metrics *metrics.Metrics
}

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func NewEventRepository(db *pgxpool.Pool, metrics *metrics.Metrics) *EventRepository {
	return &EventRepository{
	    db: db,
	    metrics: metrics,
	}
}

func (r *EventRepository) Save(ctx context.Context, event *Event) error {
    return r.SaveTx(ctx, r.db, event)
}

func (r *EventRepository) SaveTx(ctx context.Context, db Querier, event *Event) error {
	started := time.Now()
	result := "success"

	defer func() {
		r.metrics.RepositoryQueryTotal.WithLabelValues(
			"event_create",
			result,
		).Inc()

		r.metrics.RepositoryQueryDuration.WithLabelValues(
			"event_create",
			result,
		).Observe(time.Since(started).Seconds())
	}()

	const query = `
		INSERT INTO public.outbox_event (
			id,
			event_name,
			aggregate_id,
			payload,
			created_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := db.Exec(
		ctx,
		query,
		event.ID,
		event.EventName,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
	)

	return err
}