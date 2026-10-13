package usecase

import "github.com/dqso/coin-transfer-service/app/internal/entity"

type Usecase struct {
	storage entity.Storage
}

func New(storage entity.Storage) *Usecase {
	return &Usecase{
		storage: storage,
	}
}
