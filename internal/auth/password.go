package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// hashCost — стоимость bcrypt (не минимум: у bcrypt минимум равен 4).
const hashCost = 12

var (
	ErrPasswordTooLong = errors.New("password too long")
	ErrInvalidPassword = errors.New("invalid password")
)

// HashPassword считает bcrypt-хеш пароля. Для пароля длиннее допустимого
// для bcrypt возвращает ErrPasswordTooLong.
func HashPassword(password string) (string, error) {

	hash, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return "", ErrPasswordTooLong
	}

	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(hash), nil
}

// CheckPassword сверяет пароль с хешем и возвращает ErrInvalidPassword
// при несовпадении
func CheckPassword(passwordHash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrInvalidPassword
	}
	if err != nil {
		return fmt.Errorf("compare password hash: %w", err)
	}

	return nil
}
