package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func (s *Storage) CreateOrGetTransfer(ctx context.Context, idempotenceKey, fromAccountCode, toAccountCode string, amount int64) (entity.Transfer, bool, error) {
	transfer := entity.Transfer{
		IdempotencyKey: idempotenceKey,
		FromAccount:    entity.NewAccountRefAsCode(fromAccountCode),
		ToAccount:      entity.NewAccountRefAsCode(toAccountCode),
		Amount:         amount,
		Status:         entity.TransferStatusInProgress,
	}

	// If another request with the same key is in a transaction now, INSERT waits for it.
	err := s.db.QueryRow(ctx, `
		INSERT INTO transfers (idempotency_key, from_account_code, to_account_code, amount, status)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id, created_at`,
		idempotenceKey, fromAccountCode, toAccountCode, amount, transfer.Status,
	).Scan(&transfer.ID, &transfer.CreatedAt)
	if err == nil {
		return transfer, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return entity.Transfer{}, false, fmt.Errorf("insert transfer: %w", err)
	}

	saved, err := s.GetTransferByIdempotencyKey(ctx, idempotenceKey)
	if errors.Is(err, entity.ErrStorageNotFound) {
		// The other request failed and removed its transfer after our INSERT.
		return entity.Transfer{}, false, errors.New("transfer was removed, try again")
	}
	if err != nil {
		return entity.Transfer{}, false, err
	}
	return saved, false, nil
}

func (s *Storage) GetTransferByIdempotencyKey(ctx context.Context, idempotencyKey string) (entity.Transfer, error) {
	var transfer entity.Transfer
	// The account code is saved in transfers only for unknown accounts.
	// For known accounts it comes from the accounts table.
	err := s.db.QueryRow(ctx, `
		SELECT t.id,
		       t.idempotency_key,
		       t.from_account_id,
		       coalesce(fa.code, t.from_account_code),
		       t.to_account_id,
		       coalesce(ta.code, t.to_account_code),
		       t.amount,
		       t.status,
		       coalesce(t.reason_code, 0),
		       t.created_at
		FROM transfers t
		LEFT JOIN accounts fa ON fa.id = t.from_account_id
		LEFT JOIN accounts ta ON ta.id = t.to_account_id
		WHERE t.idempotency_key = $1`,
		idempotencyKey,
	).Scan(
		&transfer.ID,
		&transfer.IdempotencyKey,
		&transfer.FromAccount.ID,
		&transfer.FromAccount.Code,
		&transfer.ToAccount.ID,
		&transfer.ToAccount.Code,
		&transfer.Amount,
		&transfer.Status,
		&transfer.ReasonCode,
		&transfer.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.Transfer{}, entity.ErrStorageNotFound
	}
	if err != nil {
		return entity.Transfer{}, fmt.Errorf("get transfer by idempotency key: %w", err)
	}
	return transfer, nil
}

func (s *Storage) UpdateTransfer(ctx context.Context, transfer entity.Transfer) error {
	// For a found account we save the id and remove the code. For an unknown account we keep the code.
	tag, err := s.db.Exec(ctx, `
		UPDATE transfers
		SET status            = $2,
		    reason_code       = nullif($3::INTEGER, 0),
		    from_account_id   = $4,
		    from_account_code = CASE WHEN $4::BIGINT IS NULL THEN from_account_code END,
		    to_account_id     = $5,
		    to_account_code   = CASE WHEN $5::BIGINT IS NULL THEN to_account_code END,
		    updated_at        = now()
		WHERE id = $1`,
		transfer.ID, transfer.Status, transfer.ReasonCode, transfer.FromAccount.ID, transfer.ToAccount.ID,
	)
	if err != nil {
		return fmt.Errorf("update transfer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrStorageNotFound
	}
	return nil
}

func (s *Storage) DeleteInProgressTransfer(ctx context.Context, transferID int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM transfers WHERE id = $1 AND status = $2`, transferID, entity.TransferStatusInProgress)
	if err != nil {
		return fmt.Errorf("delete in-progress transfer: %w", err)
	}
	return nil
}
