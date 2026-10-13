package http

import (
	"context"

	"github.com/dqso/coin-transfer-service/app/internal/transport/http/oas"
)

func (h *Handler) GetAccount(ctx context.Context, params oas.GetAccountParams) (*oas.Account, error) {
	account, err := h.usecase.GetAccount(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	return &oas.Account{
		AccountID: account.Code,
		Balance:   account.Balance,
	}, nil
}
