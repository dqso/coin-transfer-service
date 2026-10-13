package entity

var (
	ErrIdempotencyKeyConflict = NewError(CodeIdempotencyKeyConflict)
	ErrTransferInProgress     = NewError(CodeTransferInProgress)
	ErrTransferNotFound       = NewError(CodeTransferNotFound)
	ErrAccountNotFound        = NewError(CodeAccountNotFound)
)

// Error is an error that the client can see. Its code tells the kind of the error.
type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// NewError creates an error with the standard message of the code.
func NewError(code Code) *Error {
	return &Error{Code: code, Message: code.String()}
}

func NewValidationError(message string) *Error {
	return &Error{Code: CodeInvalidRequest, Message: message}
}
