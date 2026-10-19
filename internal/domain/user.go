package domain

import (
	"errors"
	"time"
)

var (
	ErrEmailExists = errors.New("email already exists")
)

// UserCreate — сырой вход регистрации: пароль в открытом виде, как прислал клиент.
type UserCreate struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewUser — то, что сервис передаёт в репозиторий на вставку.
type NewUser struct {
	Email        string
	PasswordHash string
}

// User — информация о пользователе, как её отдает API.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenPair — пара токенов, которую сервис возвращает хендлеру (внутренний тип)
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// NewRefreshToken — то, что вставляем (хеш + срок считает сервис).
type NewRefreshToken struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
}

// RefreshToken — строка как она лежит в БД.
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// TokenResponse — информация про токены, как их отдает API.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
