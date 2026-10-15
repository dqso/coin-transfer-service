package worker

//go:generate go tool mockgen -typed -source=stuck_transfers.go -destination=mocks_test.go -package=worker -mock_names=usecase=MockUsecase

import (
	"context"
	"log/slog"
	"time"
)

const (
	// StuckAfter must be much longer than the time of one transfer request,
	// so the worker does not take transfers that are still in work.
	StuckAfter = time.Minute
	// stuckTransfersInterval is the pause between two runs.
	stuckTransfersInterval = 30 * time.Second
	// stuckTransfersBatch limits the work of one run.
	stuckTransfersBatch = 100
)

type usecase interface {
	FinishStuckTransfers(ctx context.Context, stuckAfter time.Duration, limit int) (int, error)
}

// StuckTransfers finishes transfers that stay in progress, for example after a crash of the service.
type StuckTransfers struct {
	log     *slog.Logger
	usecase usecase
}

func NewStuckTransfers(log *slog.Logger, usecase usecase) *StuckTransfers {
	return &StuckTransfers{
		log:     log,
		usecase: usecase,
	}
}

// Run works until the context is canceled.
func (w *StuckTransfers) Run(ctx context.Context) {
	ticker := time.NewTicker(stuckTransfersInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		finished, err := w.usecase.FinishStuckTransfers(ctx, StuckAfter, stuckTransfersBatch)
		if finished > 0 {
			w.log.InfoContext(ctx, "stuck transfers finished", slog.Int("count", finished))
		}
		if err != nil && ctx.Err() == nil {
			w.log.ErrorContext(ctx, "finish stuck transfers", slog.Any("err", err))
		}
	}
}
