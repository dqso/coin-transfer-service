//go:build loaddata

package postgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

const (
	loadAccounts       = 100_000
	loadTransfers      = 1_000_000
	loadAccountBalance = 1_000_000
)

// TestLoadData fills the database with a lot of valid data. It helps to see slow queries without indexes.
// It does not run with a normal "go test". Run it like this:
//
//	PGHOST=localhost PGUSER=app PGPASSWORD=psswrd PGDATABASE=devdb \
//	go test -tags loaddata -run TestLoadData -timeout 30m -v ./app/internal/repository/postgres/
//
// Every run adds new data with its own account prefix, so you can run it many times.
func TestLoadData(t *testing.T) {
	ctx := t.Context()

	pool, err := pgxpool.New(ctx, "")
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	prefix := fmt.Sprintf("load_%d_", time.Now().Unix())

	// All data is added in one transaction, so a failed run leaves nothing behind.
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		// Accounts get a start balance directly, like the treasury account in the seed.
		step(t, "accounts")
		_, err := tx.Exec(ctx, `
			INSERT INTO accounts (code, balance)
			SELECT $1 || n, $2
			FROM generate_series(1, $3::BIGINT) AS n`,
			prefix, loadAccountBalance, loadAccounts,
		)
		if err != nil {
			return fmt.Errorf("insert accounts: %w", err)
		}

		// The accounts of one INSERT get ids one after another, so we can pick a random account by its id.
		var firstAccountID int64
		err = tx.QueryRow(ctx, `SELECT min(id) FROM accounts WHERE code LIKE $1 || '%'`, prefix).Scan(&firstAccountID)
		if err != nil {
			return fmt.Errorf("get first account id: %w", err)
		}

		var lastTransferID int64
		err = tx.QueryRow(ctx, `SELECT coalesce(max(id), 0) FROM transfers`).Scan(&lastTransferID)
		if err != nil {
			return fmt.Errorf("get last transfer id: %w", err)
		}

		// Most transfers are completed. Some are rejected for each possible reason.
		// Amounts are small and start balances are big, so no balance goes below zero.
		step(t, "transfers")
		_, err = tx.Exec(ctx, `
			INSERT INTO transfers (
				idempotency_key,
				from_account_id, from_account_code,
				to_account_id, to_account_code,
				amount, status, reason_code, created_at, updated_at
			)
			SELECT gen_random_uuid()::TEXT,
			       CASE WHEN kind = 'unknown' THEN NULL ELSE from_id END,
			       CASE WHEN kind = 'unknown' THEN 'unknown_' || n END,
			       CASE WHEN kind = 'self' THEN from_id ELSE to_id END,
			       NULL,
			       amount,
			       (CASE WHEN kind = 'completed' THEN 'completed' ELSE 'rejected' END)::transfer_status,
			       CASE kind
			           WHEN 'insufficient' THEN $4::INTEGER
			           WHEN 'unknown' THEN $5::INTEGER
			           WHEN 'self' THEN $6::INTEGER
			       END,
			       created_at,
			       created_at
			FROM (
				SELECT n,
				       $1::BIGINT + offset_id AS from_id,
				       $1::BIGINT + (offset_id + 1 + floor(random() * ($2::BIGINT - 1))::BIGINT) % $2::BIGINT AS to_id,
				       1 + floor(random() * 10)::BIGINT AS amount,
				       now() - random() * INTERVAL '30 days' AS created_at,
				       CASE
				           WHEN r < 0.90 THEN 'completed'
				           WHEN r < 0.96 THEN 'insufficient'
				           WHEN r < 0.99 THEN 'unknown'
				           ELSE 'self'
				       END AS kind
				FROM (
					SELECT n, floor(random() * $2::BIGINT)::BIGINT AS offset_id, random() AS r
					FROM generate_series(1, $3::BIGINT) AS n
				) AS s
			) AS g`,
			firstAccountID, loadAccounts, loadTransfers,
			entity.CodeInsufficientFunds, entity.CodeTransferAccountNotFound, entity.CodeSelfTransferNotAllowed,
		)
		if err != nil {
			return fmt.Errorf("insert transfers: %w", err)
		}

		step(t, "postings")
		_, err = tx.Exec(ctx, `
			INSERT INTO postings (transfer_id, account_id, amount, created_at)
			SELECT t.id, p.account_id, p.amount, t.created_at
			FROM transfers t
			CROSS JOIN LATERAL (
				VALUES (t.from_account_id, -t.amount), (t.to_account_id, t.amount)
			) AS p (account_id, amount)
			WHERE t.id > $1 AND t.status = 'completed'`,
			lastTransferID,
		)
		if err != nil {
			return fmt.Errorf("insert postings: %w", err)
		}

		step(t, "balances")
		_, err = tx.Exec(ctx, `
			UPDATE accounts a
			SET balance = a.balance + p.total,
			    updated_at = now()
			FROM (
				SELECT p.account_id, sum(p.amount) AS total
				FROM postings p
				JOIN transfers t ON t.id = p.transfer_id
				WHERE t.id > $1
				GROUP BY p.account_id
			) AS p
			WHERE p.account_id = a.id`,
			lastTransferID,
		)
		if err != nil {
			return fmt.Errorf("update balances: %w", err)
		}

		// Transfers only move coins, so the total balance must not change.
		var total int64
		err = tx.QueryRow(ctx, `SELECT sum(balance) FROM accounts WHERE code LIKE $1 || '%'`, prefix).Scan(&total)
		if err != nil {
			return fmt.Errorf("get total balance: %w", err)
		}
		if want := int64(loadAccounts * loadAccountBalance); total != want {
			return fmt.Errorf("total balance is %d, want %d", total, want)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update the planner statistics, so query plans use the new data.
	step(t, "analyze")
	if _, err := pool.Exec(ctx, `ANALYZE accounts, transfers, postings`); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	t.Logf("done: %d accounts with prefix %q, %d transfers", loadAccounts, prefix, loadTransfers)
}

func step(t *testing.T, name string) {
	t.Helper()
	t.Logf("%s: %s", time.Now().Format(time.TimeOnly), name)
}
