package service

import (
	"context"
	"errors"
	"fmt"
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

const maxEmailLength = 255

// UserRepository - то, что сервису нужно от пользователя.
type UserRepository interface {
	CreateUser(ctx context.Context, in domain.UserCreate) (int64, error)
}

type UserService struct {
	users  UserRepository
	tokens *auth.TokenManager
}

func NewUserService(users UserRepository, tokens *auth.TokenManager) *UserService {
	return &UserService{
		users:  users,
		tokens: tokens,
	}
}

func (s *UserService) Register(ctx context.Context, in domain.UserCreate) (string, error) {
	if in.Email == "" || in.Password == "" {
		return "", ErrInvalidInput
	}
	if utf8.RuneCountInString(in.Password) > maxEmailLength {
		return "", ErrInvalidInput
	}

	passwordHash, err := auth.HashPassword(in.Password)
	if errors.Is(err, auth.ErrPasswordTooLong) {
		return "", ErrInvalidInput
	}
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	userID, err := s.users.CreateUser(ctx, domain.UserCreate{
		Email:    in.Email,
		Password: passwordHash,
	})
	if errors.Is(err, domain.ErrEmailExists) {
		return "", ErrEmailTaken
	}
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.tokens.Issue(userID)
	if err != nil {
		return "", fmt.Errorf("issue token: %w", err)
	}
	return token, nil
}
