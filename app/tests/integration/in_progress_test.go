//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insertStuckTransfer adds a transfer that looks like this: a replica took the idempotency key
// and died before it finished the transfer. Nobody will finish or delete this row.
func (a *app) insertStuckTransfer(req transferRequest) {
	a.t.Helper()
	_, err := a.pool.Exec(a.t.Context(), `
		INSERT INTO transfers (idempotency_key, from_account_code, to_account_code, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'in_progress', now() - INTERVAL '1 hour', now() - INTERVAL '1 hour')`,
		req.IdempotencyKey, req.FromAccountID, req.ToAccountID, req.Amount,
	)
	require.NoError(a.t, err, "insert stuck transfer")
}

// A stuck transfer must not block its idempotency key forever.
// The client sends the same request again and the transfer must be done.
func TestCreateTransfer_StuckInProgress(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	req := transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100}
	a.insertStuckTransfer(req)

	resp := a.postTransfer(req)

	require.Equal(t, http.StatusOK, resp.Code, resp.Message)
	assert.Equal(t, "completed", resp.Status)
	assert.EqualValues(t, 900, a.balance("acc_1"))

	// The saved transfer is the same as the answer.
	saved := a.getTransfer(req.IdempotencyKey)
	require.Equal(t, http.StatusOK, saved.Code, saved.Message)
	assert.Equal(t, "completed", saved.Status)
	assert.Equal(t, resp.TransferID, saved.TransferID)
	a.checkLedger()
}

// A stuck transfer still owns its idempotency key: a request with other details gets a conflict.
func TestCreateTransfer_StuckInProgressOtherRequest(t *testing.T) {
	t.Parallel()
	a := newApp(t)
	a.insertStuckTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 100})

	resp := a.postTransfer(transferRequest{IdempotencyKey: "key-1", FromAccountID: "acc_1", ToAccountID: "acc_2", Amount: 200})

	assert.Equal(t, http.StatusConflict, resp.Code, resp.Message)
	assert.EqualValues(t, 1000, a.balance("acc_1"))
	a.checkLedger()
}
