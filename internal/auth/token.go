package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken возвращается, когда токен повреждён, просрочен
// или подписан другим ключом.
var (
	ErrInvalidToken = errors.New("invalid token")
)

// CookieName — имя cookie, в которой передаётся токен аутентификации.
const CookieName = "token"

// Claims — полезная нагрузка токена: только стандартные поля JWT.
type Claims struct {
	jwt.RegisteredClaims
}

// TokenManager выпускает и проверяет JWT-токены, подписанные HS256.
type TokenManager struct {
	secret         []byte
	accessTokenTTL time.Duration
}

// NewTokenManager создаёт TokenManager, подписывающий токены ключом secret
// и выпускающий их со сроком жизни ttl.
func NewTokenManager(secret []byte, accessTokenTTL time.Duration) *TokenManager {
	return &TokenManager{
		secret:         secret,
		accessTokenTTL: accessTokenTTL,
	}
}

// Issue выпускает подписанный токен для пользователя userID.
func (m *TokenManager) Issue(userID int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenTTL)),
		},
	})
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tokenString, nil
}

// Verify проверяет подпись и срок действия токена и возвращает идентификатор
// пользователя. Для невалидного токена возвращает ошибку, обёрнутую ErrInvalidToken.
func (m *TokenManager) Verify(tokenString string) (int64, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, fmt.Errorf("verify token: %w: %w", ErrInvalidToken, err)
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("verify token: %w: invalid subject %q", ErrInvalidToken, claims.Subject)
	}
	return userID, nil
}
