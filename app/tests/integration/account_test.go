//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAccount_Seed(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("acc_%d", i)
		resp := a.getAccount(id)
		assert.Equal(t, http.StatusOK, resp.Code, "account %s", id)
		assert.Equal(t, id, resp.AccountID)
		assert.EqualValues(t, 1000, resp.Balance, "account %s", id)
	}
	a.checkLedger()
}

func TestGetAccount_NotFound(t *testing.T) {
	t.Parallel()
	a := newApp(t)

	resp := a.getAccount("acc_404")

	assert.Equal(t, http.StatusNotFound, resp.Code)
}
