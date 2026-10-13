package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

// querier is the common part of *pgxpool.Pool and pgx.Tx.
// The same Storage code works with both of them.
type querier interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Storage struct {
	db querier
}

var _ entity.Storage = (*Storage)(nil)

func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{db: pool}
}

// InTransaction commits the transaction if fn returns nil. Otherwise it rolls the transaction back.
// Inside a transaction it creates a savepoint.
func (s *Storage) InTransaction(ctx context.Context, fn func(ctx context.Context, tx entity.Storage) error) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		return fn(ctx, &Storage{db: tx})
	})
}
