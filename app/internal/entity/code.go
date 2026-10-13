package entity

// Code is a number for an error or for a reason of a rejected transfer.
// The first three digits are the HTTP status, the last three digits are the number inside this status.
// Codes of rejected transfers are saved in the database, so do not change the values.
type Code int

const (
	CodeNone Code = 0

	CodeInvalidRequest Code = 400001

	CodeTransferNotFound Code = 404001
	CodeAccountNotFound  Code = 404002

	CodeIdempotencyKeyConflict Code = 409001
	CodeTransferInProgress     Code = 409002

	// Reasons of rejected transfers.
	CodeSelfTransferNotAllowed  Code = 422001
	CodeTransferAccountNotFound Code = 422002
	CodeInsufficientFunds       Code = 422003

	CodeInternal Code = 500001
)

// HTTPStatus returns the HTTP status that is part of the code.
func (c Code) HTTPStatus() int {
	return int(c) / 1000
}

func (c Code) String() string {
	switch c {
	case CodeNone:
		return ""
	case CodeInvalidRequest:
		return "invalid request"
	case CodeTransferNotFound:
		return "transfer not found"
	case CodeAccountNotFound:
		return "account not found"
	case CodeIdempotencyKeyConflict:
		return "idempotency key is already used with other payment details"
	case CodeTransferInProgress:
		return "transfer is in progress"
	case CodeSelfTransferNotAllowed:
		return "self-transfer not allowed"
	case CodeTransferAccountNotFound:
		return "account not found"
	case CodeInsufficientFunds:
		return "insufficient funds"
	case CodeInternal:
		return "internal error"
	default:
		return "unknown code"
	}
}
