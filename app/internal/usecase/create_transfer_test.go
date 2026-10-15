package usecase

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func TestUsecase_CreateTransfer(t *testing.T) {
	type (
		args struct {
			idempotencyKey  string
			fromAccountCode string
			toAccountCode   string
			amount          int64
		}
		want struct {
			transfer entity.Transfer
			err      error
		}
	)

	const transferID = 10
	errStorage := errors.New("storage error")

	// newTransfer is the transfer right after CreateOrGetTransfer: it has only the account codes.
	newTransfer := func(from, to string, amount int64) entity.Transfer {
		return entity.Transfer{
			ID:             transferID,
			IdempotencyKey: "key-1",
			FromAccount:    entity.NewAccountRefAsCode(from),
			ToAccount:      entity.NewAccountRefAsCode(to),
			Amount:         amount,
			Status:         entity.TransferStatusInProgress,
		}
	}
	// finished is the transfer after executeTransfer.
	finished := func(transfer entity.Transfer, status entity.TransferStatus, code entity.Code, fromID, toID *int64) entity.Transfer {
		transfer.Status, transfer.ReasonCode = status, code
		transfer.FromAccount.ID, transfer.ToAccount.ID = fromID, toID
		return transfer
	}
	accFrom := &entity.Account{ID: 1, Code: "acc_1", Balance: 1000}
	accTo := &entity.Account{ID: 2, Code: "acc_2", Balance: 1000}

	tests := []struct {
		name    string
		args    args
		prepare func(s *MockStorage)
		want    want
	}{
		{
			name: "empty idempotency key",
			args: args{idempotencyKey: "", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			want: want{err: entity.NewValidationError("idempotency key cannot be empty")},
		},
		{
			name: "empty source account",
			args: args{idempotencyKey: "key-1", fromAccountCode: "", toAccountCode: "acc_2", amount: 100},
			want: want{err: entity.NewValidationError("source account cannot be empty")},
		},
		{
			name: "empty receiver account",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "", amount: 100},
			want: want{err: entity.NewValidationError("receiver account cannot be empty")},
		},
		{
			name: "zero amount",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 0},
			want: want{err: entity.NewValidationError("amount must be greater than zero")},
		},
		{
			name: "completed",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_2", 100)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(transferID), int64(1), int64(-100)).
						Return(nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(transferID), int64(2), int64(100)).
						Return(nil),
					s.EXPECT().UpdateAccountBalance(gomock.Any(), int64(1), int64(900)).
						Return(nil),
					s.EXPECT().UpdateAccountBalance(gomock.Any(), int64(2), int64(1100)).
						Return(nil),
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusCompleted, entity.CodeNone, new(int64(1)), new(int64(2)))).
						Return(nil),
				)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_2", 100), entity.TransferStatusCompleted,
				entity.CodeNone, new(int64(1)), new(int64(2))),
			},
		},
		{
			name: "rejected: self-transfer",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_1", amount: 100},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_1", 100)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_1", int64(100)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(transfer, nil),
					// The accounts are not locked: the rule does not need them.
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusRejected, entity.CodeSelfTransferNotAllowed, nil, nil)).
						Return(nil),
				)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_1", 100), entity.TransferStatusRejected,
				entity.CodeSelfTransferNotAllowed, nil, nil)},
		},
		{
			name: "rejected: account not found",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_404", amount: 100},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_404", 100)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_404", int64(100)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_404").
						Return(accFrom, nil, nil),
					// The found account is saved by id, the unknown one only by code.
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusRejected, entity.CodeTransferAccountNotFound, new(int64(1)), nil)).
						Return(nil),
				)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_404", 100), entity.TransferStatusRejected,
				entity.CodeTransferAccountNotFound, new(int64(1)), nil)},
		},
		{
			name: "rejected: insufficient funds",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 1001},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_2", 1001)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(1001)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusRejected, entity.CodeInsufficientFunds, new(int64(1)), new(int64(2)))).
						Return(nil),
				)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_2", 1001), entity.TransferStatusRejected,
				entity.CodeInsufficientFunds, new(int64(1)), new(int64(2)))},
		},
		{
			name: "all coins of the account",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 1000},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_2", 1000)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(1000)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(transferID), int64(1), int64(-1000)).
						Return(nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(transferID), int64(2), int64(1000)).
						Return(nil),
					s.EXPECT().UpdateAccountBalance(gomock.Any(), int64(1), int64(0)).
						Return(nil),
					s.EXPECT().UpdateAccountBalance(gomock.Any(), int64(2), int64(2000)).
						Return(nil),
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusCompleted, entity.CodeNone, new(int64(1)), new(int64(2)))).
						Return(nil),
				)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_2", 1000), entity.TransferStatusCompleted,
				entity.CodeNone, new(int64(1)), new(int64(2)))},
		},
		{
			name: "key is used: same details, saved result",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				saved := finished(newTransfer("acc_1", "acc_2", 100), entity.TransferStatusCompleted,
					entity.CodeNone, new(int64(1)), new(int64(2)))
				// No transaction: the transfer is not done again.
				s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
					Return(saved, false, nil)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_2", 100), entity.TransferStatusCompleted,
				entity.CodeNone, new(int64(1)), new(int64(2)))},
		},
		{
			name: "key is used: other details",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 200},
			prepare: func(s *MockStorage) {
				saved := finished(newTransfer("acc_1", "acc_2", 100), entity.TransferStatusCompleted,
					entity.CodeNone, new(int64(1)), new(int64(2)))
				s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(200)).
					Return(saved, false, nil)
			},
			want: want{err: entity.ErrIdempotencyKeyConflict},
		},
		{
			name: "key is used: transfer in progress",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
					Return(newTransfer("acc_1", "acc_2", 100), false, nil)
			},
			want: want{err: entity.ErrTransferInProgress},
		},
		{
			name: "worker finished the transfer while the request waited for the lock",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_2", 100)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					// The coins are not moved again.
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(finished(transfer, entity.TransferStatusCompleted, entity.CodeNone, new(int64(1)), new(int64(2))), nil),
				)
			},
			want: want{transfer: finished(newTransfer("acc_1", "acc_2", 100), entity.TransferStatusCompleted,
				entity.CodeNone, new(int64(1)), new(int64(2)))},
		},
		{
			name: "storage error on create",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
					Return(entity.Transfer{}, false, errStorage)
			},
			want: want{err: errStorage},
		},
		{
			name: "storage error in transaction: the key is freed",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_2", 100)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(transferID), int64(1), int64(-100)).
						Return(errStorage),
					s.EXPECT().DeleteInProgressTransfer(gomock.Any(), int64(transferID)).
						Return(nil),
				)
			},
			want: want{err: errStorage},
		},
		{
			name: "storage error in transaction and on delete",
			args: args{idempotencyKey: "key-1", fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			prepare: func(s *MockStorage) {
				transfer := newTransfer("acc_1", "acc_2", 100)
				gomock.InOrder(
					s.EXPECT().CreateOrGetTransfer(gomock.Any(), "key-1", "acc_1", "acc_2", int64(100)).
						Return(transfer, true, nil),
					expectInTransaction(s),
					s.EXPECT().GetTransferWithBlock(gomock.Any(), int64(transferID)).
						Return(entity.Transfer{}, errStorage),
					s.EXPECT().DeleteInProgressTransfer(gomock.Any(), int64(transferID)).
						Return(entity.ErrStorageRollback),
				)
			},
			want: want{err: errors.Join(errStorage, entity.ErrStorageRollback)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			storage := NewMockStorage(ctrl)
			if tt.prepare != nil {
				tt.prepare(storage)
			}
			u := NewUsecase(slog.New(slog.DiscardHandler), storage)

			transfer, err := u.CreateTransfer(t.Context(), tt.args.idempotencyKey, tt.args.fromAccountCode, tt.args.toAccountCode, tt.args.amount)

			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.transfer, transfer)
		})
	}
}

// expectInTransaction runs the function of the transaction with the same mock as tx.
func expectInTransaction(s *MockStorage) *MockStorageInTransactionCall {
	return s.EXPECT().InTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context, entity.Storage) error) error {
			return fn(ctx, s)
		})
}
