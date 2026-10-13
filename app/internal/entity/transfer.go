package entity

import "time"

type TransferStatus string

const (
	TransferStatusInProgress TransferStatus = "in_progress"
	TransferStatusCompleted  TransferStatus = "completed"
	TransferStatusRejected   TransferStatus = "rejected"
)

func (s TransferStatus) String() string {
	return string(s)
}

type Transfer struct {
	ID             int64
	IdempotencyKey string
	FromAccount    AccountRef
	ToAccount      AccountRef
	Amount         int64
	Status         TransferStatus
	// ReasonCode is set only for rejected transfers.
	ReasonCode Code
	CreatedAt  time.Time
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
