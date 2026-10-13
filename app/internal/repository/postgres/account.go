package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func (s *Storage) GetAccountByCode(ctx context.Context, code string) (entity.Account, error) {
	var account entity.Account
	err := s.db.QueryRow(ctx, `
		SELECT id, code, balance, created_at, updated_at
		FROM accounts
		WHERE code = $1`,
		code,
	).Scan(&account.ID, &account.Code, &account.Balance, &account.CreatedAt, &account.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.Account{}, entity.ErrStorageNotFound
	}
	if err != nil {
		return entity.Account{}, fmt.Errorf("get account by code: %w", err)
	}
	return account, nil
}

func (s *Storage) GetTwoAccountsWithBlock(ctx context.Context, fromAccountCode, toAccountCode string) (*entity.Account, *entity.Account, error) {
	// FOR UPDATE locks the rows in the ORDER BY order. Because of the same order in all
	// transactions, two opposite transfers (A to B and B to A) cannot block each other forever.
	rows, err := s.db.Query(ctx, `
		SELECT id, code, balance, created_at, updated_at
		FROM accounts
		WHERE code = ANY($1)
		ORDER BY id
		FOR UPDATE`,
		[]string{fromAccountCode, toAccountCode},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("get accounts for update: %w", err)
	}
	accounts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.Account, error) {
		var account entity.Account
		err := row.Scan(&account.ID, &account.Code, &account.Balance, &account.CreatedAt, &account.UpdatedAt)
		return account, err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("get accounts for update: %w", err)
	}

	var fromAccount, toAccount *entity.Account
	for i := range accounts {
		switch accounts[i].Code {
		case fromAccountCode:
			fromAccount = &accounts[i]
		case toAccountCode:
			toAccount = &accounts[i]
		}
	}
	return fromAccount, toAccount, nil
}

func (s *Storage) UpdateAccountBalance(ctx context.Context, accountID int64, balance int64) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE accounts
		SET balance = $2, updated_at = now()
		WHERE id = $1`,
		accountID, balance,
	)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrStorageNotFound
	}
	return nil
}
