package entity

import "time"

type TransferStatus string

const (
	TransferStatusInProgress TransferStatus = "in_progress"
	TransferStatusCompleted  TransferStatus = "completed"
	TransferStatusRejected   TransferStatus = "rejected"
)

type Transfer struct {
	ID             int64
	IdempotencyKey string
	FromAccount    AccountRef
	ToAccount      AccountRef
	Amount         int64
	Status         TransferStatus
	// Reason is set only for rejected transfers.
	Reason    string
	CreatedAt time.Time
}

func (t Transfer) IsSamePaymentDetails(fromAccountCode, toAccountCode string, amount int64) bool {
	if t.FromAccount.Code != fromAccountCode {
		return false
	}
	if t.ToAccount.Code != toAccountCode {
		return false
	}
	if t.Amount != amount {
		return false
	}
	return true
}
