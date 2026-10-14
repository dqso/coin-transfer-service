package http

import (
	"context"
	"strconv"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
	"github.com/dqso/coin-transfer-service/app/internal/transport/http/oas"
)

func (h *Handler) GetTransfer(ctx context.Context, params oas.GetTransferParams) (*oas.Transfer, error) {
	transfer, err := h.usecase.GetTransfer(ctx, params.IdempotencyKey)
	if err != nil {
		return nil, err
	}

	resp := &oas.Transfer{
		TransferID:     strconv.FormatInt(transfer.ID, 10),
		IdempotencyKey: transfer.IdempotencyKey,
		FromAccountID:  transfer.FromAccount.Code,
		ToAccountID:    transfer.ToAccount.Code,
		Amount:         transfer.Amount,
		Status:         oas.TransferStatus(transfer.Status),
		CreatedAt:      transfer.CreatedAt,
	}
	if transfer.Status == entity.TransferStatusRejected {
		resp.Reason = oas.NewOptString(transfer.ReasonCode.String())
		resp.Code = oas.NewOptInt(int(transfer.ReasonCode))
	}
	return resp, nil
}
