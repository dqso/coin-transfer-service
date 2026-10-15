package usecase

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func TestUsecase_GetAccount(t *testing.T) {
	type (
		args struct {
			accountCode string
		}
		want struct {
			account entity.Account
			err     error
		}
	)

	errStorage := errors.New("storage error")
	account := entity.Account{ID: 1, Code: "acc_1", Balance: 1000}

	tests := []struct {
		name    string
		args    args
		prepare func(s *MockStorage)
		want    want
	}{
		{
			name: "found",
			args: args{accountCode: "acc_1"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetAccountByCode(gomock.Any(), "acc_1").
					Return(account, nil)
			},
			want: want{account: account},
		},
		{
			name: "not found",
			args: args{accountCode: "acc_404"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetAccountByCode(gomock.Any(), "acc_404").
					Return(entity.Account{}, entity.ErrStorageNotFound)
			},
			want: want{err: entity.ErrAccountNotFound},
		},
		{
			name: "storage error",
			args: args{accountCode: "acc_1"},
			prepare: func(s *MockStorage) {
				s.EXPECT().GetAccountByCode(gomock.Any(), "acc_1").
					Return(entity.Account{}, errStorage)
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

			account, err := u.GetAccount(t.Context(), tt.args.accountCode)

			assert.Equal(t, tt.want.err, err)
			assert.Equal(t, tt.want.account, account)
		})
	}
}
