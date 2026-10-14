package http

import (
	"context"
	"strconv"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
	"github.com/dqso/coin-transfer-service/app/internal/transport/http/oas"
)

func (h *Handler) CreateTransfer(ctx context.Context, req *oas.CreateTransferRequest) (oas.CreateTransferRes, error) {
	transfer, err := h.usecase.CreateTransfer(ctx, req.IdempotencyKey, req.FromAccountID, req.ToAccountID, req.Amount)
	if err != nil {
		return nil, err
	}

	if transfer.Status == entity.TransferStatusRejected {
		return &oas.RejectedTransfer{
			Status: oas.RejectedTransferStatusRejected,
			Reason: transfer.ReasonCode.String(),
			Code:   int(transfer.ReasonCode),
		}, nil
	}

	return &oas.CreateTransferResponse{
		TransferID:    strconv.FormatInt(transfer.ID, 10),
		Status:        oas.CreateTransferResponseStatusCompleted,
		FromAccountID: transfer.FromAccount.Code,
		ToAccountID:   transfer.ToAccount.Code,
		Amount:        transfer.Amount,
	}, nil
}
