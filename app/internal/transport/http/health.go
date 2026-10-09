package http

import (
	"context"
	"net/http"
	"time"
)

// Liveness only shows that the app is running. It does not check the database or other services.
func (h *Handler) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Readiness shows that the app can serve requests: all readiness checks must pass.
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	for _, check := range h.readinessChecks {
		if err := check(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}
