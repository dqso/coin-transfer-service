package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccountRef(t *testing.T) {
	type want struct {
		ref AccountRef
	}

	tests := []struct {
		name string
		ref  AccountRef
		want want
	}{
		{
			name: "as code",
			ref:  NewAccountRefAsCode("acc_1"),
			want: want{ref: AccountRef{ID: nil, Code: "acc_1"}},
		},
		{
			name: "as id",
			ref:  NewAccountRefAsID(1),
			want: want{ref: AccountRef{ID: new(int64(1)), Code: ""}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want.ref, tt.ref)
		})
	}
}
