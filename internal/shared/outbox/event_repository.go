package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j_go_share_trip/internal/observability/metrics"
	"job4j_go_share_trip/internal/storage"
)

const (
	DefaultLockLimit   = 100
	MaxPublishAttempts = 10
)

type EventRepository struct {
	db      *pgxpool.Pool
	metrics *metrics.Metrics
}

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type queryExecer interface {
	Querier
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type txCtxKey struct{}

func NewEventRepository(db *pgxpool.Pool, metrics *metrics.Metrics) *EventRepository {
	return &EventRepository{
		db:      db,
		metrics: metrics,
	}
}

func (r *EventRepository) Save(ctx context.Context, event *Event) error {
	return r.SaveTx(ctx, r.db, event)
}

func (r *EventRepository) SaveTx(ctx context.Context, db Querier, event *Event) error {
	started := time.Now()
	err := r.saveTx(ctx, db, event)
	r.observe("event_create", started, err)
	return err
}

func (r *EventRepository) saveTx(ctx context.Context, db Querier, event *Event) error {
	const query = `
		INSERT INTO public.outbox_events (
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			metadata,
			status,
			attempts,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	metadata := event.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	_, err := db.Exec(
		ctx,
		query,
		event.ID,
		event.AggregateType,
		event.AggregateID,
		event.EventType,
		event.Payload,
		metadata,
		event.Status,
		event.Attempts,
		event.CreatedAt,
	)

	return err
}

func (r *EventRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return storage.TxWithoutResult(ctx, r.db, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txCtxKey{}, tx))
	})
}

func (r *EventRepository) LockPending(ctx context.Context, limit int) ([]Event, error) {
	started := time.Now()
	events, err := r.lockPending(ctx, limit)
	r.observe("event_lock_pending", started, err)
	return events, err
}

func (r *EventRepository) lockPending(ctx context.Context, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = DefaultLockLimit
	}

	const query = `
		SELECT
			id,
			aggregate_type,
			aggregate_id,
			event_type,
			payload,
			metadata,
			status,
			attempts,
			last_error,
			created_at,
			sent_at
		FROM public.outbox_events
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.querier(ctx).Query(ctx, query, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0, limit)
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.ID,
			&event.AggregateType,
			&event.AggregateID,
			&event.EventType,
			&event.Payload,
			&event.Metadata,
			&event.Status,
			&event.Attempts,
			&event.LastError,
			&event.CreatedAt,
			&event.SentAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (r *EventRepository) CountPending(ctx context.Context) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM public.outbox_events
		WHERE status = $1
	`

	var count int
	err := r.db.QueryRow(ctx, query, StatusPending).Scan(&count)
	return count, err
}

func (r *EventRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	started := time.Now()
	err := r.markSent(ctx, id)
	r.observe("event_mark_sent", started, err)
	return err
}

func (r *EventRepository) markSent(ctx context.Context, id uuid.UUID) error {
	const query = `
		UPDATE public.outbox_events
		SET
			status = $2,
			sent_at = now(),
			last_error = NULL
		WHERE id = $1
	`

	_, err := r.querier(ctx).Exec(ctx, query, id, StatusSent)
	return err
}

func (r *EventRepository) MarkFailed(ctx context.Context, id uuid.UUID, publishErr error) error {
	started := time.Now()
	err := r.markFailed(ctx, id, publishErr)
	r.observe("event_mark_failed", started, err)
	return err
}

func (r *EventRepository) markFailed(ctx context.Context, id uuid.UUID, publishErr error) error {
	lastError := ""
	if publishErr != nil {
		lastError = publishErr.Error()
	}

	const query = `
		UPDATE public.outbox_events
		SET
			attempts = attempts + 1,
			last_error = $2,
			status = CASE
				WHEN attempts + 1 >= $3 THEN $4
				ELSE $5
			END
		WHERE id = $1
	`

	_, err := r.querier(ctx).Exec(
		ctx,
		query,
		id,
		lastError,
		MaxPublishAttempts,
		StatusFailed,
		StatusPending,
	)
	return err
}

func (r *EventRepository) querier(ctx context.Context) queryExecer {
	if tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}
	return r.db
}

func (r *EventRepository) observe(operation string, started time.Time, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}

	r.metrics.RepositoryQueryTotal.WithLabelValues(operation, result).Inc()
	r.metrics.RepositoryQueryDuration.WithLabelValues(operation, result).
		Observe(time.Since(started).Seconds())
}
