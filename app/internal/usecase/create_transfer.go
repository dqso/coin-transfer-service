package usecase

import (
	"context"
	"errors"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func (u *Usecase) CreateTransfer(ctx context.Context, idempotencyKey, fromAccountCode, toAccountCode string, amount int64) (entity.Transfer, error) {
	// Validation
	if idempotencyKey == "" {
		return entity.Transfer{}, entity.ValidationError{Reason: "idempotency key cannot be empty"}
	}
	if fromAccountCode == "" {
		return entity.Transfer{}, entity.ValidationError{Reason: "source account cannot be empty"}
	}
	if toAccountCode == "" {
		return entity.Transfer{}, entity.ValidationError{Reason: "receiver account cannot be empty"}
	}
	if amount <= 0 {
		return entity.Transfer{}, entity.ValidationError{Reason: "amount must be greater than zero"}
	}

	transfer, wasCreated, err := u.storage.CreateOrGetTransfer(ctx, idempotencyKey, fromAccountCode, toAccountCode, amount)
	if err != nil {
		return entity.Transfer{}, err
	} else if !wasCreated {
		if !transfer.IsSamePaymentDetails(fromAccountCode, toAccountCode, amount) {
			return entity.Transfer{}, entity.ErrIdempotencyKeyConflict
		}
		if transfer.Status == entity.TransferStatusInProgress {
			return entity.Transfer{}, entity.ErrTransferInProgress
		}
		return transfer, nil
	}

	err = u.storage.InTransaction(ctx, func(ctx context.Context, tx entity.Storage) error {
		if err := executeTransfer(ctx, tx, &transfer); err != nil {
			return err
		}
		return tx.UpdateTransfer(ctx, transfer)
	})
	if err != nil {
		// The transaction is rolled back, so no coins were moved. Free the idempotency key,
		// so the client can send the request again. The request context can be already canceled.
		if delErr := u.storage.DeleteInProgressTransfer(context.WithoutCancel(ctx), transfer.ID); delErr != nil {
			// The transfer stays in progress. A worker will remove it later.
			return entity.Transfer{}, errors.Join(err, delErr)
		}
		return entity.Transfer{}, err
	}

	return transfer, nil
}

// executeTransfer checks the business rules and moves the coins.
// It sets the final status of the transfer. A rejected transfer is not an error.
func executeTransfer(ctx context.Context, tx entity.Storage, transfer *entity.Transfer) error {
	if transfer.FromAccount.Code == transfer.ToAccount.Code {
		transfer.Status, transfer.Reason = entity.TransferStatusRejected, "self-transfer not allowed"
		return nil
	}

	fromAccount, toAccount, err := tx.GetTwoAccountsWithBlock(ctx, transfer.FromAccount.Code, transfer.ToAccount.Code)
	if err != nil {
		return err
	}
	if fromAccount != nil {
		transfer.FromAccount.ID = new(fromAccount.ID)
	}
	if toAccount != nil {
		transfer.ToAccount.ID = new(toAccount.ID)
	}
	if fromAccount == nil || toAccount == nil {
		transfer.Status, transfer.Reason = entity.TransferStatusRejected, "account not found"
		return nil
	}

	if fromAccount.Balance < transfer.Amount {
		transfer.Status, transfer.Reason = entity.TransferStatusRejected, "insufficient funds"
		return nil
	}

	if err := tx.CreatePosting(ctx, transfer.ID, fromAccount.ID, -transfer.Amount); err != nil {
		return err
	}
	if err := tx.CreatePosting(ctx, transfer.ID, toAccount.ID, transfer.Amount); err != nil {
		return err
	}
	if err := tx.UpdateAccountBalance(ctx, fromAccount.ID, fromAccount.Balance-transfer.Amount); err != nil {
		return err
	}
	if err := tx.UpdateAccountBalance(ctx, toAccount.ID, toAccount.Balance+transfer.Amount); err != nil {
		return err
	}

	transfer.Status, transfer.Reason = entity.TransferStatusCompleted, ""
	return nil
}
