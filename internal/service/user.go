package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
)

// Ошибки регистрации и аутентификации, которые слой HTTP переводит в коды ответа.
var (
	ErrInvalidInput       = errors.New("invalid email or password format")
	ErrEmailTaken         = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid user credentials")
)

// Границы из схем RegisterRequest в api/openapi.yaml.
const (
	maxEmailLength    = 255
	minPasswordLength = 8
	maxPasswordLength = 72
)

// UserRepository - то, что сервису нужно от пользователя.
type UserRepository interface {
	CreateUser(ctx context.Context, in domain.NewUser) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}

type RefreshStore interface {
	CreateRefreshToken(ctx context.Context, in domain.NewRefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id int64) error
	RevokeAllForUser(ctx context.Context, userID int64) error
}

// TokenIssuer выпускает токен по id пользователя. Интерфейс (а не конкретный
// *auth.TokenManager) позволяет подменить выпуск токена в тестах.
type TokenIssuer interface {
	Issue(userID int64) (string, error)
}

type UserService struct {
	users           UserRepository
	tokens          TokenIssuer
	refreshTokens   RefreshStore
	refreshTokenTTL time.Duration
}

func NewUserService(users UserRepository, tokens TokenIssuer, refreshTokens RefreshStore, refreshTokenTTL time.Duration) *UserService {
	return &UserService{
		users:           users,
		tokens:          tokens,
		refreshTokens:   refreshTokens,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// normalizeEmail обрезает пробелы по краям, проверяет, что это голый email-адрес,
// и возвращает его в нижнем регистре — в том же виде, в каком уникальный индекс
// users(lower(email)) сравнивает адреса. ok=false, если адрес невалиден или это
// форма вида "Name <a@b.co>".
func normalizeEmail(email string) (string, bool) {
	trimmed := strings.TrimSpace(email)
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", false
	}
	if !strings.EqualFold(addr.Address, trimmed) {
		return "", false
	}
	return strings.ToLower(addr.Address), true
}

func (s *UserService) Register(ctx context.Context, in domain.UserCreate) (domain.User, string, error) {
	email, ok := normalizeEmail(in.Email)
	if !ok || utf8.RuneCountInString(email) > maxEmailLength {
		return domain.User{}, "", ErrInvalidInput
	}
	if len(in.Password) < minPasswordLength || len(in.Password) > maxPasswordLength {
		return domain.User{}, "", ErrInvalidInput
	}

	passwordHash, err := auth.HashPassword(in.Password)
	if errors.Is(err, auth.ErrPasswordTooLong) {
		return domain.User{}, "", ErrInvalidInput
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.CreateUser(ctx, domain.NewUser{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if errors.Is(err, domain.ErrEmailExists) {
		return domain.User{}, "", ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("issue token: %w", err)
	}
	return user, token, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (domain.TokenPair, error) {
	email, ok := normalizeEmail(email)
	if !ok || utf8.RuneCountInString(email) > maxEmailLength {
		return domain.TokenPair{}, ErrInvalidInput
	}
	if len(password) < minPasswordLength || len(password) > maxPasswordLength {
		return domain.TokenPair{}, ErrInvalidInput
	}

	user, err := s.users.GetUserByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("get user by email: %w", err)
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		return domain.TokenPair{}, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.Issue(user.ID)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("issue token: %w", err)
	}

	rawRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}
	tokenHash, err := auth.HashRefreshToken(rawRefresh)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("hash refresh token: %w", err)
	}
	if err := s.refreshTokens.CreateRefreshToken(ctx, domain.NewRefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}); err != nil {
		return domain.TokenPair{}, fmt.Errorf("store refresh token: %w", err)
	}

	return domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
	}, nil
}
