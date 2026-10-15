package usecase

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func TestUsecase_FinishStuckTransfers(t *testing.T) {
	type (
		args struct {
			stuckAfter time.Duration
			limit      int
		}
		want struct {
			finished int
			err      error
		}
	)

	const stuckAfter = time.Minute
	errStorage := errors.New("storage error")

	// stuckTransfer is a transfer in progress as the storage returns it: it has only the account codes.
	stuckTransfer := func(id, amount int64) entity.Transfer {
		return entity.Transfer{
			ID:             id,
			IdempotencyKey: "key-1",
			FromAccount:    entity.NewAccountRefAsCode("acc_1"),
			ToAccount:      entity.NewAccountRefAsCode("acc_2"),
			Amount:         amount,
			Status:         entity.TransferStatusInProgress,
		}
	}
	// finished is the transfer after executeTransfer.
	finished := func(transfer entity.Transfer, status entity.TransferStatus, code entity.Code) entity.Transfer {
		transfer.Status, transfer.ReasonCode = status, code
		transfer.FromAccount.ID, transfer.ToAccount.ID = new(int64(1)), new(int64(2))
		return transfer
	}
	accFrom := &entity.Account{ID: 1, Code: "acc_1", Balance: 1000}
	accTo := &entity.Account{ID: 2, Code: "acc_2", Balance: 1000}

	// expectNoStuckTransfers is the last transaction of a run: there is nothing to finish.
	expectNoStuckTransfers := func(s *MockStorage) []any {
		return []any{
			expectInTransaction(s),
			s.EXPECT().GetStuckTransferWithBlock(gomock.Any(), stuckAfter).
				Return(entity.Transfer{}, entity.ErrStorageNotFound),
		}
	}

	tests := []struct {
		name    string
		args    args
		prepare func(s *MockStorage)
		want    want
	}{
		{
			name: "no stuck transfers",
			args: args{stuckAfter: stuckAfter, limit: 100},
			prepare: func(s *MockStorage) {
				gomock.InOrder(expectNoStuckTransfers(s)...)
			},
			want: want{finished: 0},
		},
		{
			name: "completed",
			args: args{stuckAfter: stuckAfter, limit: 100},
			prepare: func(s *MockStorage) {
				transfer := stuckTransfer(10, 100)
				gomock.InOrder(append([]any{
					expectInTransaction(s),
					s.EXPECT().GetStuckTransferWithBlock(gomock.Any(), stuckAfter).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(10), int64(1), int64(-100)).
						Return(nil),
					s.EXPECT().CreatePosting(gomock.Any(), int64(10), int64(2), int64(100)).
						Return(nil),
					s.EXPECT().UpdateAccountBalance(gomock.Any(), int64(1), int64(900)).
						Return(nil),
					s.EXPECT().UpdateAccountBalance(gomock.Any(), int64(2), int64(1100)).
						Return(nil),
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusCompleted, entity.CodeNone)).
						Return(nil),
				}, expectNoStuckTransfers(s)...)...)
			},
			want: want{finished: 1},
		},
		{
			name: "rejected: insufficient funds",
			args: args{stuckAfter: stuckAfter, limit: 100},
			prepare: func(s *MockStorage) {
				transfer := stuckTransfer(10, 1001)
				gomock.InOrder(append([]any{
					expectInTransaction(s),
					s.EXPECT().GetStuckTransferWithBlock(gomock.Any(), stuckAfter).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					// A rejected transfer is finished too.
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusRejected, entity.CodeInsufficientFunds)).
						Return(nil),
				}, expectNoStuckTransfers(s)...)...)
			},
			want: want{finished: 1},
		},
		{
			name: "limit stops the run",
			args: args{stuckAfter: stuckAfter, limit: 1},
			prepare: func(s *MockStorage) {
				transfer := stuckTransfer(10, 1001)
				// Only one transaction: the next stuck transfer waits for the next run.
				gomock.InOrder(
					expectInTransaction(s),
					s.EXPECT().GetStuckTransferWithBlock(gomock.Any(), stuckAfter).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusRejected, entity.CodeInsufficientFunds)).
						Return(nil),
				)
			},
			want: want{finished: 1},
		},
		{
			name: "zero limit",
			args: args{stuckAfter: stuckAfter, limit: 0},
			want: want{finished: 0},
		},
		{
			name: "storage error stops the run",
			args: args{stuckAfter: stuckAfter, limit: 100},
			prepare: func(s *MockStorage) {
				transfer := stuckTransfer(10, 1001)
				gomock.InOrder(
					expectInTransaction(s),
					s.EXPECT().GetStuckTransferWithBlock(gomock.Any(), stuckAfter).
						Return(transfer, nil),
					s.EXPECT().GetTwoAccountsWithBlock(gomock.Any(), "acc_1", "acc_2").
						Return(accFrom, accTo, nil),
					s.EXPECT().UpdateTransfer(gomock.Any(), finished(transfer, entity.TransferStatusRejected, entity.CodeInsufficientFunds)).
						Return(nil),
					// The first transfer is already saved, so it is counted.
					expectInTransaction(s),
					s.EXPECT().GetStuckTransferWithBlock(gomock.Any(), stuckAfter).
						Return(entity.Transfer{}, errStorage),
				)
			},
			want: want{finished: 1, err: errStorage},
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

			finished, err := u.FinishStuckTransfers(t.Context(), tt.args.stuckAfter, tt.args.limit)

			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.finished, finished)
		})
	}
}
