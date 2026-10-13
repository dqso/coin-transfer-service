package usecase

import (
	"context"
	"errors"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func (u *Usecase) GetTransfer(ctx context.Context, idempotencyKey string) (entity.Transfer, error) {
	transfer, err := u.storage.GetTransferByIdempotencyKey(ctx, idempotencyKey)
	if errors.Is(err, entity.ErrStorageNotFound) {
		return entity.Transfer{}, entity.ErrTransferNotFound
	}
	if err != nil {
		return entity.Transfer{}, err
	}

	// The API has only final statuses. The client must ask again later.
	if transfer.Status == entity.TransferStatusInProgress {
		return entity.Transfer{}, entity.ErrTransferInProgress
	}

	return transfer, nil
}
