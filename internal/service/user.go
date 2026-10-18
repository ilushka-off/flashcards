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

const (
	minEmailLength = 8
	maxEmailLength = 254
)

// UserRepository - то, что сервису нужно от пользователя.
type UserRepository interface {
	CreateUser(ctx context.Context, in domain.UserCreate) (domain.User, error)
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

func (s *UserService) Register(ctx context.Context, in domain.UserCreate) (domain.User, string, error) {
	if in.Email == "" || in.Password == "" {
		return domain.User{}, "", ErrInvalidInput
	}
	if utf8.RuneCountInString(in.Email) > maxEmailLength || utf8.RuneCountInString(in.Email) < minEmailLength {
		return domain.User{}, "", ErrInvalidInput
	}

	passwordHash, err := auth.HashPassword(in.Password)
	if errors.Is(err, auth.ErrPasswordTooLong) {
		return domain.User{}, "", ErrInvalidInput
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.CreateUser(ctx, domain.UserCreate{
		Email:    in.Email,
		Password: passwordHash,
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
