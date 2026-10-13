//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// response has the fields of all API answers. Each answer fills only its own fields.
type response struct {
	Code int `json:"-"`

	TransferID     string    `json:"transfer_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	FromAccountID  string    `json:"from_account_id"`
	ToAccountID    string    `json:"to_account_id"`
	Amount         int64     `json:"amount"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`

	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`

	Message string `json:"message"`
}

type transferRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	FromAccountID  string `json:"from_account_id"`
	ToAccountID    string `json:"to_account_id"`
	Amount         int64  `json:"amount"`
}

func (a *app) postTransfer(req transferRequest) response {
	a.t.Helper()
	body, err := json.Marshal(req)
	if !assert.NoError(a.t, err, "marshal request") {
		return response{}
	}
	return a.do(http.MethodPost, "/api/v1/transfers", body)
}

func (a *app) getTransfer(idempotencyKey string) response {
	a.t.Helper()
	return a.do(http.MethodGet, "/api/v1/transfers/"+idempotencyKey, nil)
}

func (a *app) getAccount(id string) response {
	a.t.Helper()
	return a.do(http.MethodGet, "/api/v1/accounts/"+id, nil)
}

// balance returns the balance from the API. The account must exist.
func (a *app) balance(id string) int64 {
	a.t.Helper()
	resp := a.getAccount(id)
	require.Equal(a.t, http.StatusOK, resp.Code, "get account %s: %s", id, resp.Message)
	return resp.Balance
}

// do is safe to call from many goroutines. It uses assert, not require, for this reason:
// require stops the test, and this works only in the goroutine of the test.
func (a *app) do(method, path string, body []byte) response {
	a.t.Helper()
	req, err := http.NewRequestWithContext(a.t.Context(), method, a.url+path, bytes.NewReader(body))
	if !assert.NoError(a.t, err, "create request") {
		return response{}
	}
	req.Header.Set("Content-Type", "application/json")

	// The log is shown for failed tests, and for all tests with -v.
	a.t.Logf("--> %s %s %s", method, path, body)

	httpResp, err := http.DefaultClient.Do(req)
	if !assert.NoError(a.t, err, "%s %s", method, path) {
		return response{}
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if !assert.NoError(a.t, err, "%s %s: read response", method, path) {
		return response{}
	}
	a.t.Logf("<-- %d %s", httpResp.StatusCode, bytes.TrimSpace(respBody))

	resp := response{Code: httpResp.StatusCode}
	assert.NoError(a.t, json.Unmarshal(respBody, &resp), "%s %s: decode response", method, path)
	return resp
}
