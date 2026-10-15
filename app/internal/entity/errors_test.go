package entity

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	type (
		args struct {
			err error
		}
		want struct {
			code    Code
			message string
		}
	)

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "standard message of the code",
			args: args{err: NewError(CodeTransferNotFound)},
			want: want{code: CodeTransferNotFound, message: "transfer not found"},
		},
		{
			name: "validation error",
			args: args{err: NewValidationError("amount must be greater than zero")},
			want: want{code: CodeInvalidRequest, message: "amount must be greater than zero"},
		},
		{
			// The transport layer finds the error inside a wrapped one.
			name: "wrapped error",
			args: args{err: fmt.Errorf("get transfer: %w", ErrTransferInProgress)},
			want: want{code: CodeTransferInProgress, message: "transfer is in progress"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityErr, ok := errors.AsType[*Error](tt.args.err)

			require.True(t, ok, "the error must be *Error")
			assert.Equal(t, tt.want.code, entityErr.Code)
			assert.Equal(t, tt.want.message, entityErr.Error())
		})
	}
}
