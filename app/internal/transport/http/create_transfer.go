package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type CreateTransferInput struct {
	Body struct {
		IdempotencyKey string `json:"idempotency_key" minLength:"1"`
		FromAccountID  string `json:"from_account_id" minLength:"1"`
		ToAccountID    string `json:"to_account_id" minLength:"1"`
		Amount         int64  `json:"amount"`
	}
}

type CreateTransferOutput struct {
	Body struct {
		TransferID    string `json:"transfer_id"`
		Status        string `json:"status"`
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		Amount        int64  `json:"amount"`
	}
}

func (h *Handler) CreateTransfer(_ context.Context, _ *CreateTransferInput) (*CreateTransferOutput, error) {
	return nil, huma.Error501NotImplemented("not implemented")
}
