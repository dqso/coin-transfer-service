package http

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type GetTransferInput struct {
	IdempotencyKey string `path:"idempotency_key"`
}

type GetTransferOutput struct {
	Body struct {
		TransferID     string    `json:"transfer_id"`
		IdempotencyKey string    `json:"idempotency_key"`
		FromAccountID  string    `json:"from_account_id"`
		ToAccountID    string    `json:"to_account_id"`
		Amount         int64     `json:"amount"`
		Status         string    `json:"status"`
		CreatedAt      time.Time `json:"created_at"`
	}
}

func (h *Handler) GetTransfer(_ context.Context, _ *GetTransferInput) (*GetTransferOutput, error) {
	return nil, huma.Error501NotImplemented("not implemented")
}
