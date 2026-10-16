package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// healthTimeout ограничивает проверку базы в /health,
// чтобы зависшая база не подвешивала сам эндпоинт.
const healthTimeout = 2 * time.Second

type pinger interface {
	Ping(context.Context) error
}

// HealthHandler обслуживает проверку живости: отвечает о доступности сервиса и базы данных.
type HealthHandler struct {
	db     pinger
	logger *slog.Logger
}

// NewHealthHandler создаёт HealthHandler поверх pinger (обычно пула БД) и логгера.
func NewHealthHandler(db pinger, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		db:     db,
		logger: logger,
	}
}

// Check пингует базу в пределах healthTimeout и отвечает 200 со status "ok",
// либо 503 со status "unavailable", если база не ответила.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), healthTimeout)
	defer cancel()

	status, code := "ok", http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		h.logger.WarnContext(ctx, "database unavailable", "err", err)
		status, code = "unavailable", http.StatusServiceUnavailable
	}

	WriteJSON(w, code, map[string]string{"status": status})
}
