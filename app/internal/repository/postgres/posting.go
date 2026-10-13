package postgres

import (
	"context"
	"fmt"
)

func (s *Storage) CreatePosting(ctx context.Context, transferID int64, accountID int64, amount int64) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO postings (transfer_id, account_id, amount)
		VALUES ($1, $2, $3)`,
		transferID, accountID, amount,
	)
	if err != nil {
		return fmt.Errorf("create posting: %w", err)
	}
	return nil
}
