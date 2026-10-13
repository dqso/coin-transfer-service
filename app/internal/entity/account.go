package entity

import "time"

type Account struct {
	ID        int64
	Code      string
	Balance   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AccountRef points to an account from a transfer.
// ID is nil if the client sent an unknown account.
// Code is always set: it comes from the account or from the request.
type AccountRef struct {
	ID   *int64
	Code string
}

func NewAccountRefAsCode(code string) AccountRef {
	return AccountRef{
		Code: code,
	}
}

func NewAccountRefAsID(id int64) AccountRef {
	return AccountRef{
		ID: &id,
	}
}
