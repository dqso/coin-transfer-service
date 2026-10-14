//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dqso/coin-transfer-service/app/internal/worker"
)

// insertInProgressTransfer adds a transfer that was started age ago and is not finished.
// With a big age it looks like this: a replica took the idempotency key and died before it finished the transfer.
func (a *app) insertInProgressTransfer(req transferRequest, age time.Duration) {
	a.t.Helper()
	_, err := a.pool.Exec(a.t.Context(), `
		INSERT INTO transfers (idempotency_key, from_account_code, to_account_code, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'in_progress', now() - make_interval(secs => $5), now() - make_interval(secs => $5))`,
		req.IdempotencyKey, req.FromAccountID, req.ToAccountID, req.Amount, age.Seconds(),
	)
	require.NoError(a.t, err, "insert in-progress transfer")
}

// finishStuckTransfers does one run of the stuck transfers worker.
func (a *app) finishStuckTransfers() int {
	a.t.Helper()
	finished, err := a.usecase.FinishStuckTransfers(a.t.Context(), worker.StuckAfter, 100)
	require.NoError(a.t, err, "finish stuck transfers")
	return finished
}

// A stuck transfer must not block its idempotency key forever.
// The worker finishes it, and the client gets the result with the same request.
func TestFinishStuckTransfers_Completed(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}
	a.insertInProgressTransfer(req, time.Hour)

	// Before the worker the transfer is still in progress.
	resp := a.postTransfer(req)
	assert.Equal(t, http.StatusConflict, resp.Code, resp.Message)
	assert.EqualValues(t, 1000, a.balance("acc_1"))

	assert.Equal(t, 1, a.finishStuckTransfers())

	resp = a.postTransfer(req)
	require.Equal(t, http.StatusOK, resp.Code, resp.Message)
	assert.Equal(t, "completed", resp.Status)
	assert.EqualValues(t, 900, a.balance("acc_1"))
	assert.EqualValues(t, 1100, a.balance("acc_2"))

	// The saved transfer is the same as the answer.
	saved := a.getTransfer(req.IdempotencyKey)
	require.Equal(t, http.StatusOK, saved.Code, saved.Message)
	assert.Equal(t, "completed", saved.Status)
	assert.Equal(t, resp.TransferID, saved.TransferID)
	a.checkLedger()
}

// The worker checks the same business rules as a new request.
func TestFinishStuckTransfers_Rejected(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 5000}
	a.insertInProgressTransfer(req, time.Hour)

	assert.Equal(t, 1, a.finishStuckTransfers())

	resp := a.postTransfer(req)
	require.Equal(t, http.StatusUnprocessableEntity, resp.Code, resp.Message)
	assert.Equal(t, "rejected", resp.Status)
	assert.Equal(t, "insufficient funds", resp.Reason)
	assert.EqualValues(t, 1000, a.balance("acc_1"))
	a.checkLedger()
}

// A new transfer in progress can be in work in another replica, so the worker does not take it.
func TestFinishStuckTransfers_SkipsFresh(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}
	a.insertInProgressTransfer(req, 0)

	assert.Equal(t, 0, a.finishStuckTransfers())

	saved := a.getTransfer(req.IdempotencyKey)
	assert.Equal(t, http.StatusConflict, saved.Code, saved.Message)
	assert.EqualValues(t, 1000, a.balance("acc_1"))
}

// A stuck transfer still owns its idempotency key: a request with other details gets a conflict.
func TestCreateTransfer_StuckInProgressOtherRequest(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	a.insertInProgressTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}, time.Hour)

	resp := a.postTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 200})

	assert.Equal(t, http.StatusConflict, resp.Code, resp.Message)
	assert.EqualValues(t, 1000, a.balance("acc_1"))
	a.checkLedger()
}
