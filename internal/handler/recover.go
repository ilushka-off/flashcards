package handler

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover ловит panic из нижележащих обработчиков, пишет один Error со stack
// trace (request_id добавляет ContextHandler) и отвечает 500 в формате ошибок API.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				// http.ErrAbortHandler — штатный сигнал прервать обработчик, а не сбой;
				// пробрасываем его дальше, чтобы сервер отработал как задумано.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				logger.ErrorContext(r.Context(), "panic recovered",
					"err", rec,
					"stack", string(debug.Stack()))
				WriteError(w, http.StatusInternalServerError, CodeInternal, msgInternal)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
