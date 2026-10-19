package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ilushka-off/flashcards/internal/userctx"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Кладём пустой holder в начале цепочки: auth-middleware ниже заполнит
			// его userID, и после раскрутки мы увидим его здесь (контекст сам по
			// себе течёт только вниз).
			r = r.WithContext(userctx.NewContext(r.Context()))

			lw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lw, r)

			duration := time.Since(start)

			// InfoContext — чтобы обёртка ContextHandler подставила request_id.
			attrs := []any{
				"method", r.Method,
				"route", chi.RouteContext(r.Context()).RoutePattern(),
				"path", r.URL.Path,
				"duration", duration.Seconds(),
				"http_status_code", lw.statusCode,
			}
			if userID, ok := userctx.UserIDFrom(r.Context()); ok {
				attrs = append(attrs, "user_id", userID)
			}
			log.InfoContext(r.Context(), "HTTP request info", attrs...)
		})
	}
}
