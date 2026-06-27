package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitOfWork struct {
    db  *pgxpool.Pool
    tx  pgx.Tx
    ctx context.Context
}

func NewUnitOfWork(db *pgxpool.Pool) *UnitOfWork {
    return &UnitOfWork{db: db}
}

func (uow *UnitOfWork) Begin(ctx context.Context) error {
    tx, err := uow.db.Begin(ctx)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    uow.tx = tx
    uow.ctx = ctx
    return nil
}

func (uow *UnitOfWork) Commit() error {
    if uow.tx == nil {
        return errors.New("no transaction in progress")
    }
    return uow.tx.Commit(uow.ctx)
}

func (uow *UnitOfWork) Rollback() error {
    if uow.tx == nil {
        return errors.New("no transaction in progress")
    }
    return uow.tx.Rollback(uow.ctx)
}

func (uow *UnitOfWork) GetTx() pgx.Tx {
    return uow.tx
}