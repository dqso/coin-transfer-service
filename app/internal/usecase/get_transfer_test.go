package usecase

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func TestUsecase_GetTransfer(t *testing.T) {
	type (
		args struct {
			idempotencyKey string
		}
		want struct {
			transfer entity.Transfer
			err      error
		}
	)

	errStorage := errors.New("storage error")

	newTransfer := func(status entity.TransferStatus, code entity.Code) entity.Transfer {
		return entity.Transfer{
			ID:             10,
			IdempotencyKey: "key-1",
			FromAccount:    entity.AccountRef{ID: new(int64(1)), Code: "acc_1"},
			ToAccount:      entity.AccountRef{ID: new(int64(2)), Code: "acc_2"},
			Amount:         100,
			Status:         status,
			ReasonCode:     code,
		}
	}

	tests := []struct {
		name    string
		args    args
		prepare func(s *MockStorage)
		want    want
	}{
		{
			name: "completed",
			args: args{idempotencyKey: "key-1"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetTransferByIdempotencyKey(gomock.Any(), "key-1").
					Return(newTransfer(entity.TransferStatusCompleted, entity.CodeNone), nil)
			},
			want: want{transfer: newTransfer(entity.TransferStatusCompleted, entity.CodeNone)},
		},
		{
			name: "rejected",
			args: args{idempotencyKey: "key-1"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetTransferByIdempotencyKey(gomock.Any(), "key-1").
					Return(newTransfer(entity.TransferStatusRejected, entity.CodeInsufficientFunds), nil)
			},
			want: want{transfer: newTransfer(entity.TransferStatusRejected, entity.CodeInsufficientFunds)},
		},
		{
			name: "in progress",
			args: args{idempotencyKey: "key-1"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetTransferByIdempotencyKey(gomock.Any(), "key-1").
					Return(newTransfer(entity.TransferStatusInProgress, entity.CodeNone), nil)
			},
			want: want{err: entity.ErrTransferInProgress},
		},
		{
			name: "not found",
			args: args{idempotencyKey: "key-404"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetTransferByIdempotencyKey(gomock.Any(), "key-404").
					Return(entity.Transfer{}, entity.ErrStorageNotFound)
			},
			want: want{err: entity.ErrTransferNotFound},
		},
		{
			name: "storage error",
			args: args{idempotencyKey: "key-1"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetTransferByIdempotencyKey(gomock.Any(), "key-1").
					Return(entity.Transfer{}, errStorage)
			},
			want: want{err: errStorage},
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

			transfer, err := u.GetTransfer(t.Context(), tt.args.idempotencyKey)

			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.transfer, transfer)
		})
	}
}
