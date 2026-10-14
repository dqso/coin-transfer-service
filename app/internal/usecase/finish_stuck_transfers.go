package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

// FinishStuckTransfers finishes transfers that stay in progress for longer than stuckAfter.
// A transfer gets stuck if the service stops in the middle of the work, for example it crashes.
// The transfer is done in the same way as for a new request, so the client gets the result by the same key.
// It finishes up to limit transfers and returns how many it finished.
// Several replicas can run it at the same time: each transfer is taken by one of them.
func (u *Usecase) FinishStuckTransfers(ctx context.Context, stuckAfter time.Duration, limit int) (int, error) {
	finished := 0
	for finished < limit {
		// One transaction for one transfer: an error in one transfer does not undo the others.
		var transfer entity.Transfer
		found := false
		err := u.storage.InTransaction(ctx, func(ctx context.Context, tx entity.Storage) error {
			var err error
			transfer, err = tx.GetStuckTransferWithBlock(ctx, stuckAfter)
			if errors.Is(err, entity.ErrStorageNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			found = true

			if err := executeTransfer(ctx, tx, &transfer); err != nil {
				return err
			}
			return tx.UpdateTransfer(ctx, transfer)
		})
		if err != nil {
			return finished, err
		}
		if !found {
			break
		}
		u.logTransfer(ctx, "stuck transfer executed", transfer)
		finished++
	}
	return finished, nil
}
