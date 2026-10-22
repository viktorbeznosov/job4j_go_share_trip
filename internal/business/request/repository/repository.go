package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"job4j_go_share_trip/internal/observability/metrics"
)

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type RowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type RequestRepository struct {
	db      *pgxpool.Pool
	metrics *metrics.Metrics
}

func NewPostgresRepository(db *pgxpool.Pool, metrics *metrics.Metrics) *RequestRepository {
	return &RequestRepository{
		db:      db,
		metrics: metrics,
	}
}

func (r *RequestRepository) GetDB() *pgxpool.Pool {
	return r.db
}
