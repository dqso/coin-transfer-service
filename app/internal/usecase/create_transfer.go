package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

// transferTimeout limits the work of the first request with an idempotency key.
// Other requests with the same key wait for this work, so it must not depend on one client.
const transferTimeout = 10 * time.Second

func (u *Usecase) CreateTransfer(ctx context.Context, idempotencyKey, fromAccountCode, toAccountCode string, amount int64) (entity.Transfer, error) {
	// Validation
	if idempotencyKey == "" {
		return entity.Transfer{}, entity.NewValidationError("idempotency key cannot be empty")
	}
	if fromAccountCode == "" {
		return entity.Transfer{}, entity.NewValidationError("source account cannot be empty")
	}
	if toAccountCode == "" {
		return entity.Transfer{}, entity.NewValidationError("receiver account cannot be empty")
	}
	if amount <= 0 {
		return entity.Transfer{}, entity.NewValidationError("amount must be greater than zero")
	}

	// The first request with this key does the work. If the client of the first request goes away,
	// the work must go on for the other requests, so it does not use the request context.
	result := u.transfersInFlight.DoChan(idempotencyKey, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), transferTimeout)
		defer cancel()
		return u.createOrGetTransfer(ctx, idempotencyKey, fromAccountCode, toAccountCode, amount)
	})

	var transfer entity.Transfer
	select {
	case <-ctx.Done():
		return entity.Transfer{}, ctx.Err()
	case res := <-result:
		if res.Err != nil {
			return entity.Transfer{}, res.Err
		}
		transfer = res.Val.(entity.Transfer)
	}

	// Requests in one group can have different details, so every request checks them itself.
	if !transfer.IsSamePaymentDetails(fromAccountCode, toAccountCode, amount) {
		return entity.Transfer{}, entity.ErrIdempotencyKeyConflict
	}
	// Another instance of the service can still do this transfer,
	// or the transfer is stuck and waits for the stuck transfers worker.
	if transfer.Status == entity.TransferStatusInProgress {
		return entity.Transfer{}, entity.ErrTransferInProgress
	}
	return transfer, nil
}

// createOrGetTransfer does the transfer if the idempotency key is new.
// If the key is already used, it returns the saved transfer as it is.
func (u *Usecase) createOrGetTransfer(ctx context.Context, idempotencyKey, fromAccountCode, toAccountCode string, amount int64) (entity.Transfer, error) {
	transfer, wasCreated, err := u.storage.CreateOrGetTransfer(ctx, idempotencyKey, fromAccountCode, toAccountCode, amount)
	if err != nil {
		return entity.Transfer{}, err
	}
	if !wasCreated {
		return transfer, nil
	}

	err = u.storage.InTransaction(ctx, func(ctx context.Context, tx entity.Storage) error {
		// The stuck transfers worker can take this transfer if the request is very slow.
		// The lock and the status check make sure that only one of them does the transfer.
		locked, err := tx.GetTransferWithBlock(ctx, transfer.ID)
		if err != nil {
			return err
		}
		if locked.Status != entity.TransferStatusInProgress {
			transfer = locked
			return nil
		}

		if err := executeTransfer(ctx, tx, &transfer); err != nil {
			return err
		}
		return tx.UpdateTransfer(ctx, transfer)
	})
	if err != nil {
		// The transaction is rolled back, so no coins were moved. Free the idempotency key,
		// so the client can send the request again. The request context can be already canceled.
		if delErr := u.storage.DeleteInProgressTransfer(context.WithoutCancel(ctx), transfer.ID); delErr != nil {
			// The transfer stays in progress. The stuck transfers worker will finish it later.
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
		transfer.Status, transfer.ReasonCode = entity.TransferStatusRejected, entity.CodeSelfTransferNotAllowed
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
		transfer.Status, transfer.ReasonCode = entity.TransferStatusRejected, entity.CodeTransferAccountNotFound
		return nil
	}

	if fromAccount.Balance < transfer.Amount {
		transfer.Status, transfer.ReasonCode = entity.TransferStatusRejected, entity.CodeInsufficientFunds
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

	transfer.Status, transfer.ReasonCode = entity.TransferStatusCompleted, entity.CodeNone
	return nil
}
