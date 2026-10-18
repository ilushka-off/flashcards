package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/service"
)

// UserHandler обслуживает регистрацию и аутентификацию пользователей.
type UserHandler struct {
	userService  *service.UserService
	tokenTTL     time.Duration
	cookieSecure bool
	logger       *slog.Logger
}

// NewUserHandler создаёт UserHandler поверх UserService. tokenTTL должен совпадать
// со сроком жизни токена у TokenManager, чтобы кука и токен истекали одновременно.
func NewUserHandler(userService *service.UserService, tokenTTL time.Duration, cookieSecure bool, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService:  userService,
		tokenTTL:     tokenTTL,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

func (h *UserHandler) setAuthToken(w http.ResponseWriter, token string) {
	w.Header().Add("Authorization", "Bearer "+token)
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.tokenTTL.Seconds()),
	})
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.UserCreate

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректное тело запроса")
		return
	}
	user, token, err := h.userService.Register(r.Context(), req)
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректный email или пароль")
		return
	case errors.Is(err, service.ErrEmailTaken):
		WriteError(w, http.StatusConflict, CodeConflict, "email: уже зарегистрирован")
		return
	case err != nil:
		WriteError(w, http.StatusInternalServerError, CodeInternal, msgInternal)
		return
	}
	h.setAuthToken(w, token)
	WriteJSON(w, http.StatusCreated, user)
}
