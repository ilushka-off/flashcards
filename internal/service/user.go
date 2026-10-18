package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
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

// validEmail проверяет, что строка — это голый email-адрес.
// ParseAddress принимает и формы вида "Name <a@b.co>", поэтому
// дополнительно требуем, чтобы распознанный адрес совпадал с вводом.
func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func (s *UserService) Register(ctx context.Context, in domain.UserCreate) (domain.User, string, error) {
	if utf8.RuneCountInString(in.Email) > maxEmailLength || !validEmail(in.Email) {
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
