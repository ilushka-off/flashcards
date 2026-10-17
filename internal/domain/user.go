package domain

import "time"

// UserCreate — данные для создания пользователя.
type UserCreate struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// User — информация о пользователе, как её отдает API.
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
