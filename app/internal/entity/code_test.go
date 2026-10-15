package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCode(t *testing.T) {
	type (
		args struct {
			code Code
		}
		want struct {
			httpStatus int
			text       string
		}
	)

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "none", args: args{code: CodeNone}, want: want{httpStatus: 0, text: ""}},
		{name: "invalid request", args: args{code: CodeInvalidRequest}, want: want{httpStatus: 400, text: "invalid request"}},
		{name: "transfer not found", args: args{code: CodeTransferNotFound}, want: want{httpStatus: 404, text: "transfer not found"}},
		{name: "account not found", args: args{code: CodeAccountNotFound}, want: want{httpStatus: 404, text: "account not found"}},
		{
			name: "idempotency key conflict",
			args: args{code: CodeIdempotencyKeyConflict},
			want: want{httpStatus: 409, text: "idempotency key is already used with other payment details"},
		},
		{name: "transfer in progress", args: args{code: CodeTransferInProgress}, want: want{httpStatus: 409, text: "transfer is in progress"}},
		{
			name: "self-transfer not allowed",
			args: args{code: CodeSelfTransferNotAllowed},
			want: want{httpStatus: 422, text: "self-transfer not allowed"},
		},
		{
			name: "transfer account not found",
			args: args{code: CodeTransferAccountNotFound},
			want: want{httpStatus: 422, text: "account not found"},
		},
		{name: "insufficient funds", args: args{code: CodeInsufficientFunds}, want: want{httpStatus: 422, text: "insufficient funds"}},
		{name: "internal", args: args{code: CodeInternal}, want: want{httpStatus: 500, text: "internal error"}},
		{name: "unknown", args: args{code: Code(418001)}, want: want{httpStatus: 418, text: "unknown code"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.httpStatus, tt.args.code.HTTPStatus())
			assert.Equal(t, tt.want.text, tt.args.code.String())
		})
	}
}

// The codes of rejected transfers are saved in the database, so they must never change.
func TestCode_SavedValues(t *testing.T) {
	assert.Equal(t, Code(422001), CodeSelfTransferNotAllowed)
	assert.Equal(t, Code(422002), CodeTransferAccountNotFound)
	assert.Equal(t, Code(422003), CodeInsufficientFunds)
}
