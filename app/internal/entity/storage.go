package entity

import (
	"context"
	"errors"
	"time"
)

type Storage interface {
	InTransaction(ctx context.Context, fn func(ctx context.Context, tx Storage) error) error
	// CreateOrGetTransfer creates an in-progress transfer. If the idempotency key is already used, it returns the saved transfer.
	CreateOrGetTransfer(ctx context.Context, idempotenceKey, fromAccountCode, toAccountCode string, amount int64) (_ Transfer, wasCreated bool, _ error)
	GetTransferByIdempotencyKey(ctx context.Context, idempotencyKey string) (Transfer, error)
	// GetTransferWithBlock locks the transfer row. It waits if another transaction holds the lock.
	GetTransferWithBlock(ctx context.Context, transferID int64) (Transfer, error)
	// GetStuckTransferWithBlock locks one transfer that is in progress for longer than stuckAfter.
	// Locked transfers are skipped. It returns ErrStorageNotFound if there are no such transfers.
	GetStuckTransferWithBlock(ctx context.Context, stuckAfter time.Duration) (Transfer, error)
	// UpdateTransfer saves the status, the reason code and the account ids of the transfer.
	UpdateTransfer(ctx context.Context, transfer Transfer) error
	GetAccountByCode(ctx context.Context, code string) (Account, error)
	// GetTwoAccountsWithBlock locks the accounts in the order of their ids. An account is nil if it is not found.
	GetTwoAccountsWithBlock(ctx context.Context, fromAccountCode, toAccountCode string) (*Account, *Account, error)
	UpdateAccountBalance(ctx context.Context, accountID int64, balance int64) error
	CreatePosting(ctx context.Context, transferID int64, accountID int64, amount int64) error
	DeleteInProgressTransfer(ctx context.Context, transferID int64) error
}

var (
	ErrStorageRollback = errors.New("rollback")
	ErrStorageNotFound = errors.New("not found")
)
