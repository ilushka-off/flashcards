package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync/atomic"
)

// RequestIDHeader — заголовок, через который приходит и возвращается идентификатор запроса.
const RequestIDHeader = "X-Request-ID"

// maxRequestIDLen ограничивает длину принимаемого извне идентификатора, чтобы
// клиент не раздувал логи и заголовки ответа произвольной строкой.
const maxRequestIDLen = 128

// contextKey — приватный тип ключей контекста, чтобы не пересекаться с чужими строками.
type contextKey int

const requestIDKey contextKey = iota

// RequestID гарантирует, что у каждого запроса есть идентификатор: берёт его из
// заголовка X-Request-ID или генерирует новый, кладёт в контекст и возвращает
// клиенту в заголовке ответа.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID(id) {
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

// validRequestID принимает только непустой идентификатор разумной длины из
// безопасных ASCII-символов; всё остальное отбрасывается и генерируется свой.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '-' || c == '_' || c == '.':
		default:
			return false
		}
	}
	return true
}

// fallbackCounter даёт уникальный суффикс, если crypto/rand недоступен, чтобы
// такие запросы всё равно можно было различать в логах.
var fallbackCounter atomic.Uint64

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand практически не падает; на всякий случай выдаём уникальный маркер,
		// а не одинаковый для всех запросов.
		return "fallback-" + strconv.FormatUint(fallbackCounter.Add(1), 10)
	}
	return hex.EncodeToString(b[:])
}
