package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/service"
	"github.com/ilushka-off/flashcards/internal/userctx"
)

// UserRegistrar — то, что хендлеру нужно от сервиса пользователей.
type UserRegistrar interface {
	Register(ctx context.Context, in domain.UserCreate) (domain.User, string, error)
	Login(ctx context.Context, email, password string) (domain.TokenPair, error)
	GetUserByID(ctx context.Context, userID int64) (domain.User, error)
}

// UserHandler обслуживает регистрацию и аутентификацию пользователей.
type UserHandler struct {
	userService  UserRegistrar
	tokenTTL     time.Duration
	cookieSecure bool
	logger       *slog.Logger
}

// NewUserHandler создаёт UserHandler поверх UserService. tokenTTL должен совпадать
// со сроком жизни токена у TokenManager, чтобы кука и токен истекали одновременно.
func NewUserHandler(userService UserRegistrar, tokenTTL time.Duration, cookieSecure bool, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService:  userService,
		tokenTTL:     tokenTTL,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

// setAuthToken кладёт токен в httpOnly-куку. Authorization — заголовок запроса,
// сервер его в ответе не возвращает; клиент берёт токен из куки.
func (h *UserHandler) setAuthToken(w http.ResponseWriter, token string) {
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

	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, h.logger, err, "")
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
		writeServiceError(w, r, h.logger, err, "")
		return
	}
	h.setAuthToken(w, token)
	WriteJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.UserCreate

	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}

	pair, err := h.userService.Login(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректный email или пароль")
		return
	case errors.Is(err, service.ErrInvalidCredentials):
		WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "неверный email или пароль")
		return
	case err != nil:
		writeServiceError(w, r, h.logger, err, "")
		return
	}
	WriteJSON(w, http.StatusOK, domain.TokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.tokenTTL.Seconds()),
	})
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userctx.UserIDFrom(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "требуется авторизация")
		return
	}
	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "требуется авторизация")
		return
	}
	WriteJSON(w, http.StatusOK, user)
}
