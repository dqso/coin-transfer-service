package http

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type GetAccountInput struct {
	ID string `path:"id"`
}

type GetAccountOutput struct {
	Body struct {
		AccountID string `json:"account_id"`
		Balance   int64  `json:"balance"`
	}
}

func (h *Handler) GetAccount(_ context.Context, _ *GetAccountInput) (*GetAccountOutput, error) {
	return nil, huma.Error501NotImplemented("not implemented")
}
