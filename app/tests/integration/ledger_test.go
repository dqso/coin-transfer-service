//go:build integration

package integration

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// totalCoins is the sum of all balances after the seed. Transfers only move coins, so it never changes.
const totalCoins = 10_000

// checkLedger checks the rules of the ledger directly in the database.
// Call it at the end of a test, after all requests are done.
func (a *app) checkLedger() {
	a.t.Helper()
	ctx := a.t.Context()

	var total int64
	err := a.pool.QueryRow(ctx, `SELECT sum(balance) FROM accounts`).Scan(&total)
	require.NoError(a.t, err, "get total balance")
	assert.EqualValues(a.t, totalCoins, total, "total balance")

	var unbalancedTransfers int
	err = a.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM (
			SELECT transfer_id
			FROM postings
			GROUP BY transfer_id
			HAVING sum(amount) <> 0 OR count(*) <> 2
		) AS t`,
	).Scan(&unbalancedTransfers)
	require.NoError(a.t, err, "check postings")
	assert.Zero(a.t, unbalancedTransfers, "transfers with postings that do not add up to zero")

	// The treasury got its coins without a posting, so it is not checked here.
	var wrongBalances int
	err = a.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM accounts a
		WHERE a.code <> 'treasury'
		  AND a.balance <> coalesce((SELECT sum(p.amount) FROM postings p WHERE p.account_id = a.id), 0)`,
	).Scan(&wrongBalances)
	require.NoError(a.t, err, "check balances")
	assert.Zero(a.t, wrongBalances, "accounts with a balance that is not equal to the sum of their postings")

	var completedWithoutPostings int
	err = a.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM transfers t
		WHERE t.status = 'completed'
		  AND NOT EXISTS (SELECT 1 FROM postings p WHERE p.transfer_id = t.id)`,
	).Scan(&completedWithoutPostings)
	require.NoError(a.t, err, "check completed transfers")
	assert.Zero(a.t, completedWithoutPostings, "completed transfers without postings")
}
