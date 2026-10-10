package http

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// ReadinessCheck returns an error if something we need (for example, the database) does not work.
type ReadinessCheck func(ctx context.Context) error

type Handler struct {
	mux             *http.ServeMux
	readinessChecks []ReadinessCheck
}

func NewHandler(readinessChecks ...ReadinessCheck) *Handler {
	h := &Handler{
		mux:             http.NewServeMux(),
		readinessChecks: readinessChecks,
	}

	h.mux.HandleFunc("GET /livez", h.Liveness)
	h.mux.HandleFunc("GET /readyz", h.Readiness)

	api := humago.New(h.mux, huma.DefaultConfig("Coin Transfer Service", "1.0.0"))

	huma.Register(api, huma.Operation{
		OperationID: "create-transfer",
		Method:      http.MethodPost,
		Path:        "/api/v1/transfers",
		Summary:     "Transfer coins between accounts",
	}, h.CreateTransfer)

	huma.Register(api, huma.Operation{
		OperationID: "get-transfer",
		Method:      http.MethodGet,
		Path:        "/api/v1/transfers/{idempotency_key}",
		Summary:     "Get a transfer by idempotency key",
	}, h.GetTransfer)

	huma.Register(api, huma.Operation{
		OperationID: "get-account",
		Method:      http.MethodGet,
		Path:        "/api/v1/accounts/{id}",
		Summary:     "Get account balance",
	}, h.GetAccount)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
