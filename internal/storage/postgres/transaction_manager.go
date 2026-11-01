package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TransactionManager struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

type contextKey struct{}

var transactionKey contextKey

func NewTransactionManager(pool *pgxpool.Pool, queryTimeout time.Duration) *TransactionManager {
	return &TransactionManager{pool: pool, queryTimeout: queryTimeout}
}

func (m *TransactionManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := fromContext(ctx); ok {
		return fn(ctx)
	}

	beginContext, cancel := context.WithTimeout(ctx, m.queryTimeout)
	dbTx, err := m.pool.BeginTx(beginContext, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	cancel()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			rollbackContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.queryTimeout)
			_ = dbTx.Rollback(rollbackContext)
			cancel()
		}
	}()

	txContext := context.WithValue(ctx, transactionKey, dbTx)
	if err := fn(txContext); err != nil {
		return err
	}

	commitContext, cancel := context.WithTimeout(ctx, m.queryTimeout)
	commitErr := dbTx.Commit(commitContext)
	cancel()
	if commitErr != nil {
		return fmt.Errorf("commit transaction: %w", commitErr)
	}

	committed = true
	return nil
}

func executor(ctx context.Context, fallback DBTX) DBTX {
	if dbTx, ok := fromContext(ctx); ok {
		return dbTx
	}
	return fallback
}

func fromContext(ctx context.Context) (pgx.Tx, bool) {
	dbTx, ok := ctx.Value(transactionKey).(pgx.Tx)
	return dbTx, ok
}
