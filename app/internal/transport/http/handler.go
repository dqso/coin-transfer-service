package http

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/dqso/coin-transfer-service/app/internal/entity"
	"github.com/dqso/coin-transfer-service/app/internal/transport/http/oas"
)

//go:generate go tool ogen --config ogen.yml --target oas --package oas --clean openapi.yaml

//go:embed openapi.yaml
var openapiSpec []byte

type Handler struct {
	log             *slog.Logger
	mux             *http.ServeMux
	root            http.Handler
	usecase         usecase
	readinessChecks []ReadinessCheck
}

var _ oas.Handler = (*Handler)(nil)

type (
	usecase interface {
		CreateTransfer(ctx context.Context,
			idempotencyKey, fromAccountCode, toAccountCode string, amount int64,
		) (entity.Transfer, error)
		GetAccount(ctx context.Context, accountCode string) (entity.Account, error)
		GetTransfer(ctx context.Context, idempotencyKey string) (entity.Transfer, error)
	}

	// ReadinessCheck returns an error if something we need (for example, the database) does not work.
	ReadinessCheck func(ctx context.Context) error
)

func NewHandler(log *slog.Logger, usecase usecase, readinessChecks ...ReadinessCheck) (*Handler, error) {
	h := &Handler{
		log:             log,
		mux:             http.NewServeMux(),
		usecase:         usecase,
		readinessChecks: readinessChecks,
	}

	api, err := oas.NewServer(h, oas.WithErrorHandler(h.handleRequestError))
	if err != nil {
		return nil, err
	}

	h.mux.HandleFunc("GET /livez", h.Liveness)
	h.mux.HandleFunc("GET /readyz", h.Readiness)
	h.mux.HandleFunc("GET /openapi.yaml", h.OpenAPISpec)
	h.mux.HandleFunc("GET /docs", h.Docs)
	h.mux.Handle("/", api)
	h.root = h.withRequestLog(h.mux)

	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.root.ServeHTTP(w, r)
}

func (h *Handler) OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openapiSpec)
}

// NewError turns an error from a handler into an HTTP response.
func (h *Handler) NewError(ctx context.Context, err error) *oas.ErrorStatusCode {
	if entityErr, ok := errors.AsType[*entity.Error](err); ok {
		return newErrorResponse(entityErr.Code, entityErr.Message)
	}

	// Do not show internal errors to the client.
	h.log.ErrorContext(ctx, "request failed", slog.Any("err", err))
	return newErrorResponse(entity.CodeInternal, entity.CodeInternal.String())
}

// handleRequestError answers when the request cannot be read, for example the body is not valid.
// It uses the same error format as NewError.
func (h *Handler) handleRequestError(ctx context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	resp := newErrorResponse(entity.CodeInvalidRequest, err.Error())
	if ogenerrors.ErrorCode(err) >= http.StatusInternalServerError {
		h.log.ErrorContext(ctx, "request failed", slog.Any("err", err))
		resp = newErrorResponse(entity.CodeInternal, entity.CodeInternal.String())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_ = json.NewEncoder(w).Encode(resp.Response)
}

// newErrorResponse takes the HTTP status from the code.
func newErrorResponse(code entity.Code, message string) *oas.ErrorStatusCode {
	return &oas.ErrorStatusCode{
		StatusCode: code.HTTPStatus(),
		Response:   oas.Error{Code: int(code), Message: message},
	}
}
