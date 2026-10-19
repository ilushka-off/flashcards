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

// dummyPasswordHash — заранее посчитанный bcrypt-хеш той же стоимости. Считается
// один раз при инициализации пакета, чтобы сверка пароля для несуществующего
// пользователя занимала столько же времени, сколько для существующего.
var dummyPasswordHash = mustHashDummy()

func mustHashDummy() string {
	h, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-real-password"), hashCost)
	if err != nil {
		panic(fmt.Sprintf("auth: precompute dummy password hash: %v", err))
	}
	return string(h)
}

// DummyPasswordCheck тратит примерно столько же времени, сколько CheckPassword,
// но всегда безуспешен. Вызывается в ветке «пользователь не найден», чтобы по
// времени ответа нельзя было отличить существующий email от несуществующего.
func DummyPasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
}

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
