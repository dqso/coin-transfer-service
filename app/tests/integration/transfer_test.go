//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTransfer_Completed(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	resp := a.postTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100})

	require.Equal(t, http.StatusOK, resp.Code, resp.Message)
	assert.Equal(t, "completed", resp.Status)
	assert.NotEmpty(t, resp.TransferID)
	assert.Equal(t, "acc_1", resp.FromAccountID)
	assert.Equal(t, "acc_2", resp.ToAccountID)
	assert.EqualValues(t, 100, resp.Amount)
	assert.EqualValues(t, 900, a.balance("acc_1"))
	assert.EqualValues(t, 1100, a.balance("acc_2"))
	a.checkLedger()
}

func TestCreateTransfer_AllCoins(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	// The balance can become exactly zero.
	resp := a.postTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 1000})

	require.Equal(t, http.StatusOK, resp.Code, resp.Message)
	assert.EqualValues(t, 0, a.balance("acc_1"))
	a.checkLedger()
}

func TestCreateTransfer_Rejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		req    transferRequest
		reason string
	}{
		{
			name:   "insufficient funds",
			req:    transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 1001},
			reason: "insufficient funds",
		},
		{
			name:   "self-transfer",
			req:    transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_1", Amount: 100},
			reason: "self-transfer not allowed",
		},
		{
			name:   "unknown source account",
			req:    transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_404", ToAccountID: "acc_2", Amount: 100},
			reason: "account not found",
		},
		{
			name:   "unknown receiver account",
			req:    transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_404", Amount: 100},
			reason: "account not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newApp(t)

			resp := a.postTransfer(tt.req)

			require.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Message)
			assert.Equal(t, "rejected", resp.Status)
			assert.Equal(t, tt.reason, resp.Reason)
			assert.EqualValues(t, 1000, a.balance("acc_1"))
			assert.EqualValues(t, 1000, a.balance("acc_2"))
			a.checkLedger()
		})
	}
}

func TestCreateTransfer_Validation(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	tests := []struct {
		name string
		req  transferRequest
	}{
		{name: "empty idempotency key", req: transferRequest{IdempotencyKey: "", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}},
		{name: "empty source account", req: transferRequest{IdempotencyKey: "key-1", FromAccountID: "", ToAccountID: "acc_2", Amount: 100}},
		{name: "empty receiver account", req: transferRequest{IdempotencyKey: "key-2", FromAccountID: "acc_1", ToAccountID: "", Amount: 100}},
		{name: "zero amount", req: transferRequest{IdempotencyKey: "key-3", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 0}},
		{name: "negative amount", req: transferRequest{IdempotencyKey: "key-4", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: -100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := a.postTransfer(tt.req)
			assert.Equal(t, http.StatusBadRequest, resp.Code)
		})
	}

	// A request with a validation error does not take the idempotency key.
	resp := a.postTransfer(transferRequest{IdempotencyKey: "key-3", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100})
	assert.Equal(t, http.StatusOK, resp.Code, "valid request after a validation error: %s", resp.Message)
	a.checkLedger()
}

func TestCreateTransfer_SameKeySameRequest(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}

	first := a.postTransfer(req)
	second := a.postTransfer(req)

	require.Equal(t, http.StatusOK, first.Code, first.Message)
	assert.Equal(t, first, second, "the second response must be the same as the first")
	// The transfer is applied only once.
	assert.EqualValues(t, 900, a.balance("acc_1"))
	a.checkLedger()
}

func TestCreateTransfer_SameKeyAfterRejection(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 1500}

	first := a.postTransfer(req)
	require.Equal(t, http.StatusUnprocessableEntity, first.Code, first.Message)

	// Now acc_1 has enough coins, but the repeated request must get the saved answer.
	topUp := a.postTransfer(transferRequest{IdempotencyKey: "key-2", FromAccountID: "acc_3", ToAccountID: "acc_1", Amount: 1000})
	require.Equal(t, http.StatusOK, topUp.Code, topUp.Message)

	second := a.postTransfer(req)

	assert.Equal(t, first, second, "the second response must be the same as the first")
	assert.EqualValues(t, 2000, a.balance("acc_1"))
	a.checkLedger()
}

func TestCreateTransfer_SameKeyOtherRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		first  transferRequest
		second transferRequest
	}{
		{
			name:   "other amount",
			first:  transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100},
			second: transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 200},
		},
		{
			name:   "other receiver",
			first:  transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100},
			second: transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_3", Amount: 100},
		},
		{
			name:   "first request had an unknown account",
			first:  transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_404", ToAccountID: "acc_2", Amount: 100},
			second: transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100},
		},
		{
			name:   "first request was a self-transfer",
			first:  transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_1", Amount: 100},
			second: transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := newApp(t)
			a.postTransfer(tt.first)
			balanceBefore := a.balance("acc_1")

			resp := a.postTransfer(tt.second)

			assert.Equal(t, http.StatusConflict, resp.Code, resp.Message)
			assert.Equal(t, balanceBefore, a.balance("acc_1"))
			a.checkLedger()
		})
	}
}

func TestCreateTransfer_ParallelSameKey(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}

	const requests = 20
	responses := make([]response, requests)
	var wg sync.WaitGroup
	for i := range requests {
		wg.Go(func() {
			responses[i] = a.postTransfer(req)
		})
	}
	wg.Wait()

	require.Equal(t, http.StatusOK, responses[0].Code, responses[0].Message)
	for i, resp := range responses {
		assert.Equal(t, responses[0], resp, "response %d must be the same as the first", i)
	}
	assert.EqualValues(t, 900, a.balance("acc_1"))
	a.checkLedger()
}

func TestCreateTransfer_ParallelSpending(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	// acc_1 has 1000 coins, so only 10 of 50 transfers of 100 coins can pass.
	const requests = 50
	codes := make([]int, requests)
	var wg sync.WaitGroup
	for i := range requests {
		wg.Go(func() {
			codes[i] = a.postTransfer(transferRequest{
				IdempotencyKey: fmt.Sprintf("key-%d", i),
				FromAccountID:  "acc_1",
				ToAccountID:    fmt.Sprintf("acc_%d", 2+i%9),
				Amount:         100,
			}).Code
		})
	}
	wg.Wait()

	completed, rejected := 0, 0
	for _, code := range codes {
		switch code {
		case http.StatusOK:
			completed++
		case http.StatusUnprocessableEntity:
			rejected++
		default:
			assert.Fail(t, "unexpected status", "status %d", code)
		}
	}
	assert.Equal(t, 10, completed, "completed transfers")
	assert.Equal(t, requests-10, rejected, "rejected transfers")
	assert.EqualValues(t, 0, a.balance("acc_1"))
	a.checkLedger()
}

func TestCreateTransfer_ParallelOppositeTransfers(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	// Transfers in both directions lock the same two accounts.
	// They must not block each other forever.
	const requests = 40
	var wg sync.WaitGroup
	for i := range requests {
		wg.Go(func() {
			from, to := "acc_1", "acc_2"
			if i%2 == 1 {
				from, to = to, from
			}
			resp := a.postTransfer(transferRequest{IdempotencyKey: fmt.Sprintf("key-%d", i), FromAccountID: from, ToAccountID: to, Amount: 10})
			assert.Equal(t, http.StatusOK, resp.Code, "transfer %d: %s", i, resp.Message)
		})
	}
	wg.Wait()

	// The same number of transfers went each way.
	assert.EqualValues(t, 1000, a.balance("acc_1"))
	assert.EqualValues(t, 1000, a.balance("acc_2"))
	a.checkLedger()
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	created := a.postTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100})
	require.Equal(t, http.StatusOK, created.Code, created.Message)
	rejected := a.postTransfer(transferRequest{IdempotencyKey: "key-2", FromAccountID: "acc_404", ToAccountID: "acc_2", Amount: 100})
	require.Equal(t, http.StatusUnprocessableEntity, rejected.Code, rejected.Message)

	t.Run("completed", func(t *testing.T) {
		resp := a.getTransfer("key-1")

		require.Equal(t, http.StatusOK, resp.Code, resp.Message)
		assert.Equal(t, created.TransferID, resp.TransferID)
		assert.Equal(t, "key-1", resp.IdempotencyKey)
		assert.Equal(t, "acc_1", resp.FromAccountID)
		assert.Equal(t, "acc_2", resp.ToAccountID)
		assert.EqualValues(t, 100, resp.Amount)
		assert.Equal(t, "completed", resp.Status)
		assert.False(t, resp.CreatedAt.IsZero(), "created_at must be set")
	})

	t.Run("rejected with unknown account", func(t *testing.T) {
		resp := a.getTransfer("key-2")

		require.Equal(t, http.StatusOK, resp.Code, resp.Message)
		// The unknown account code comes back as the client sent it.
		assert.Equal(t, "acc_404", resp.FromAccountID)
		assert.Equal(t, "rejected", resp.Status)
	})

	t.Run("not found", func(t *testing.T) {
		resp := a.getTransfer("key-404")

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}
