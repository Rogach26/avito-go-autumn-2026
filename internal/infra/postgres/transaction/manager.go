package transaction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Manager struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	logger       *slog.Logger
}

type contextKey struct{}

func NewManager(pool *pgxpool.Pool, queryTimeout time.Duration, logger *slog.Logger) *Manager {
	return &Manager{pool: pool, queryTimeout: queryTimeout, logger: logger}
}

func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := FromContext(ctx); ok {
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
		if committed {
			return
		}

		rollbackContext, rollbackCancel := context.WithTimeout(context.WithoutCancel(ctx), m.queryTimeout)
		defer rollbackCancel()
		if rollbackErr := dbTx.Rollback(rollbackContext); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			m.logger.WarnContext(rollbackContext, "transaction rollback failed", "error", rollbackErr)
		}
	}()

	txContext := context.WithValue(ctx, contextKey{}, dbTx)
	if err := fn(txContext); err != nil {
		return err
	}

	commitContext, commitCancel := context.WithTimeout(ctx, m.queryTimeout)
	commitErr := dbTx.Commit(commitContext)
	commitCancel()
	if commitErr != nil {
		return fmt.Errorf("commit transaction: %w", commitErr)
	}

	committed = true
	return nil
}

func FromContext(ctx context.Context) (pgx.Tx, bool) {
	dbTx, ok := ctx.Value(contextKey{}).(pgx.Tx)
	return dbTx, ok
}
