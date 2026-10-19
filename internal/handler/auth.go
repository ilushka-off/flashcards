package handler

import (
	"net/http"
	"strings"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/userctx"
)

func tokenFromRequest(r *http.Request) string {
	headerValue := r.Header.Get("Authorization")
	if value, ok := strings.CutPrefix(headerValue, "Bearer "); ok {
		return value
	}
	return ""
}

// Auth возвращает middleware, пропускающее только запросы с валидным токеном
// в заголовке Authorization. Идентификатор пользователя кладётся
// в контекст запроса, откуда его достаёт UserIDFromContext. Остальным запросам
// отвечает 401.
func Auth(tokens *auth.TokenManager) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := tokenFromRequest(r)
			if token == "" {
				unauthorized(w)
				return
			}
			userID, err := tokens.Verify(token)
			if err != nil {
				unauthorized(w)
				return
			}
			ctx := userctx.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "неверный токен авторизации")
}
