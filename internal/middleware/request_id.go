package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// RequestIDHeader — заголовок, через который приходит и возвращается идентификатор запроса.
const RequestIDHeader = "X-Request-ID"

// contextKey — приватный тип ключей контекста, чтобы не пересекаться с чужими строками.
type contextKey int

const requestIDKey contextKey = iota

// RequestID гарантирует, что у каждого запроса есть идентификатор: берёт его из
// заголовка X-Request-ID или генерирует новый, кладёт в контекст и возвращает
// клиенту в заголовке ответа.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext возвращает идентификатор запроса из контекста или "", если его нет.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand практически не падает; в этом случае лучше любой маркер, чем пусто.
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}
