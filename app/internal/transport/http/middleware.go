package http

import (
	"crypto/rand"
	"log/slog"
	"net/http"
	"time"

	"github.com/dqso/coin-transfer-service/app/internal/logger"
)

const (
	requestIDHeader    = "X-Request-ID"
	maxRequestIDLength = 128
)

// withRequestLog gives every request an id and writes one log record when the request is done.
// The client can send its own id in the X-Request-ID header. The id is sent back in the same header.
func (h *Handler) withRequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = rand.Text()
		}
		w.Header().Set(requestIDHeader, requestID)
		ctx := logger.WithRequestID(r.Context(), requestID)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))

		// Probes are called every few seconds, their logs are only noise.
		if r.URL.Path == "/livez" || r.URL.Path == "/readyz" {
			return
		}
		h.log.InfoContext(ctx, "request done",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.status),
			slog.String("duration", time.Since(start).String()),
		)
	})
}

// isValidRequestID accepts a short id with visible ASCII characters only,
// so a client cannot put anything strange into the logs.
func isValidRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maxRequestIDLength {
		return false
	}
	for i := 0; i < len(requestID); i++ {
		if requestID[i] < '!' || requestID[i] > '~' {
			return false
		}
	}
	return true
}

// statusRecorder remembers the status of the response for the log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
