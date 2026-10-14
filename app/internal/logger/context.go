package logger

import (
	"context"
	"log/slog"
)

type requestIDKey struct{}

// WithRequestID saves the request id in the context. ContextHandler adds it to every log record.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// ContextHandler adds values from the context to log records.
// Use the *Context methods of the logger (InfoContext and others) to pass the context.
type ContextHandler struct {
	slog.Handler
}

func NewContextHandler(handler slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: handler}
}

func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := RequestID(ctx); requestID != "" {
		record.AddAttrs(slog.String("request_id", requestID))
	}
	return h.Handler.Handle(ctx, record)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}
