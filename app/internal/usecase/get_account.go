package usecase

import (
	"context"
	"errors"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

func (u *Usecase) GetAccount(ctx context.Context, accountCode string) (entity.Account, error) {
	account, err := u.storage.GetAccountByCode(ctx, accountCode)
	if errors.Is(err, entity.ErrStorageNotFound) {
		return entity.Account{}, entity.ErrAccountNotFound
	}
	if err != nil {
		return entity.Account{}, err
	}

	return account, nil
}
