package usecase

import (
	"golang.org/x/sync/singleflight"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
)

type Usecase struct {
	storage entity.Storage
	// transfersInFlight joins requests with the same idempotency key:
	// only the first one does the work, the others wait and get its result.
	transfersInFlight singleflight.Group
}

func NewUsecase(storage entity.Storage) *Usecase {
	return &Usecase{
		storage: storage,
	}
}
