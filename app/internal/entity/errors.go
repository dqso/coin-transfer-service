package entity

import "errors"

var (
	ErrIdempotencyKeyConflict = errors.New("idempotency key is already used with other payment details")
	ErrTransferInProgress     = errors.New("transfer is in progress")
	ErrTransferNotFound       = errors.New("transfer not found")
	ErrAccountNotFound        = errors.New("account not found")
)

type ValidationError struct {
	Reason string
}

func (e ValidationError) Error() string {
	return e.Reason
}
