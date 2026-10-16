package middleware

import (
	"context"
	"log/slog"
)

// ContextHandler оборачивает slog.Handler и добавляет в каждую запись request_id
// из контекста. Благодаря этому в коде достаточно вызывать slog c *Context
// (например logger.InfoContext(ctx, ...)), а идентификатор подставляется сам.
type ContextHandler struct {
	slog.Handler
}

// NewContextHandler оборачивает h так, чтобы записи несли request_id из контекста.
func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: h}
}

// Handle добавляет request_id из контекста, если он есть, и передаёт запись дальше.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs и WithGroup переопределены, чтобы производные хендлеры тоже
// оставались обёрнутыми и продолжали добавлять request_id.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{Handler: h.Handler.WithGroup(name)}
}
