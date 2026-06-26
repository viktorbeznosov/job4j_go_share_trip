package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Tx[T interface{}](
	ctx context.Context,
	pool *pgxpool.Pool,
	block func(tx pgx.Tx) (*T, error),
) (*T, error) {
	txBegin, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := txBegin.Rollback(ctx); rollbackErr != nil {
				// Логируем ошибку rollback, но возвращаем основную ошибку
				log.Printf("failed to rollback transaction: %v (original error: %v)", rollbackErr, err)
			}
		}
	}()

	res, err := block(txBegin)
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	if err = txBegin.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return res, nil
}

func TxWithoutResult(
	ctx context.Context,
	pool *pgxpool.Pool,
	block func(tx pgx.Tx) error,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				// Логируем ошибку rollback, но возвращаем основную ошибку
				log.Printf("failed to rollback transaction: %v (original error: %v)", rollbackErr, err)
			}
		}
	}()

	if err = block(tx); err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}