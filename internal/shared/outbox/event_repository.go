package outbox

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct {
	db *pgxpool.Pool
}

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func NewEventRepository(db *pgxpool.Pool) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Save(ctx context.Context, event *Event) error {
    return r.SaveTx(ctx, r.db, event)
}

func (r *EventRepository) SaveTx(ctx context.Context, db Querier, event *Event) error {
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