package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransfer_IsSamePaymentDetails(t *testing.T) {
	type (
		args struct {
			fromAccountCode string
			toAccountCode   string
			amount          int64
		}
		want struct {
			isSame bool
		}
	)

	// The saved transfer has a known source account and an unknown receiver account.
	transfer := Transfer{
		FromAccount: AccountRef{ID: new(int64(1)), Code: "acc_1"},
		ToAccount:   NewAccountRefAsCode("acc_404"),
		Amount:      100,
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "same details",
			args: args{fromAccountCode: "acc_1", toAccountCode: "acc_404", amount: 100},
			want: want{isSame: true},
		},
		{
			name: "other source account",
			args: args{fromAccountCode: "acc_2", toAccountCode: "acc_404", amount: 100},
			want: want{isSame: false},
		},
		{
			name: "other receiver account",
			args: args{fromAccountCode: "acc_1", toAccountCode: "acc_2", amount: 100},
			want: want{isSame: false},
		},
		{
			name: "other amount",
			args: args{fromAccountCode: "acc_1", toAccountCode: "acc_404", amount: 200},
			want: want{isSame: false},
		},
		{
			name: "accounts in the other order",
			args: args{fromAccountCode: "acc_404", toAccountCode: "acc_1", amount: 100},
			want: want{isSame: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSame := transfer.IsSamePaymentDetails(tt.args.fromAccountCode, tt.args.toAccountCode, tt.args.amount)

			assert.Equal(t, tt.want.isSame, isSame)
		})
	}
}

func TestTransferStatus_String(t *testing.T) {
	// The statuses are saved in the database and sent in the API, so they must never change.
	assert.Equal(t, "in_progress", TransferStatusInProgress.String())
	assert.Equal(t, "completed", TransferStatusCompleted.String())
	assert.Equal(t, "rejected", TransferStatusRejected.String())
}
