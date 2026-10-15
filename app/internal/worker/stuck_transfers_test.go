package worker

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestStuckTransfers_Run(t *testing.T) {
	type (
		// result is what FinishStuckTransfers returns in one run.
		result struct {
			finished int
			err      error
		}
		args struct {
			// results has one item for every run of the worker.
			results []result
			// stopInLastRun cancels the context inside the last run, like a shutdown in the middle of the work.
			stopInLastRun bool
		}
		want struct {
			logs []string
		}
	)

	errStorage := errors.New("storage error")

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "no runs before the interval",
			args: args{},
			want: want{},
		},
		{
			name: "nothing to finish",
			args: args{results: []result{{finished: 0}}},
			want: want{},
		},
		{
			name: "finished transfers",
			args: args{results: []result{{finished: 3}}},
			want: want{logs: []string{
				`level=INFO msg="stuck transfers finished" count=3`,
			}},
		},
		{
			name: "error",
			args: args{results: []result{{err: errStorage}}},
			want: want{logs: []string{
				`level=ERROR msg="finish stuck transfers" err="storage error"`,
			}},
		},
		{
			name: "some transfers are finished before the error",
			args: args{results: []result{{finished: 2, err: errStorage}}},
			want: want{logs: []string{
				`level=INFO msg="stuck transfers finished" count=2`,
				`level=ERROR msg="finish stuck transfers" err="storage error"`,
			}},
		},
		{
			name: "the worker goes on after an error",
			args: args{results: []result{{err: errStorage}, {finished: 1}}},
			want: want{logs: []string{
				`level=ERROR msg="finish stuck transfers" err="storage error"`,
				`level=INFO msg="stuck transfers finished" count=1`,
			}},
		},
		{
			name: "shutdown in the middle of the run is not an error",
			args: args{results: []result{{finished: 1, err: context.Canceled}}, stopInLastRun: true},
			want: want{logs: []string{
				`level=INFO msg="stuck transfers finished" count=1`,
			}},
		},
	}
	for _, tt := range tests {
		// synctest uses a fake clock: time.Sleep and the ticker of the worker do not wait for real.
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				ctrl := gomock.NewController(t)
				uc := NewMockUsecase(ctrl)
				calls := make([]any, 0, len(tt.args.results))
				for i, res := range tt.args.results {
					call := uc.EXPECT().FinishStuckTransfers(gomock.Any(), StuckAfter, stuckTransfersBatch)
					if tt.args.stopInLastRun && i == len(tt.args.results)-1 {
						call.DoAndReturn(func(context.Context, time.Duration, int) (int, error) {
							cancel()
							return res.finished, res.err
						})
					} else {
						call.Return(res.finished, res.err)
					}
					calls = append(calls, call)
				}
				gomock.InOrder(calls...)

				var logs bytes.Buffer
				w := NewStuckTransfers(newTestLogger(&logs), uc)
				done := make(chan struct{})
				go func() {
					w.Run(ctx)
					close(done)
				}()

				// Wait for all runs, but stop before the next one.
				time.Sleep(time.Duration(len(tt.args.results))*stuckTransfersInterval + time.Second)
				synctest.Wait()
				cancel()
				<-done

				assert.Equal(t, tt.want.logs, logLines(&logs))
			})
		})
	}
}

// newTestLogger writes logs without time, so they can be compared as text.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 && attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	}))
}

func logLines(buf *bytes.Buffer) []string {
	text := strings.TrimSpace(buf.String())
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}
