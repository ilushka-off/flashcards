package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Форматы вывода логов для LOG_FORMAT.
const (
	LogFormatJSON = "json"
	LogFormatText = "text"
)

// Config — параметры запуска сервиса.
type Config struct {
	HTTPPort     int
	DatabaseURI  string
	LogLevel     slog.Level
	LogFormat    string
	JWTSecret    string
	CookieSecure bool
}

// Ошибки конфигурации, при которых сервис не может быть запущен.
var (
	ErrDatabaseURINotFound = errors.New("database uri not found")
	ErrJWTSecretNotFound   = errors.New("jwt secret not found")
)

const minJWTSecretLength = 32

// Load собирает конфигурацию из переменных окружения
// HTTP_PORT, DATABASE_URI, LOG_LEVEL, LOG_FORMAT, JWT_SECRET, COOKIE_SECURE.
// DATABASE_URI и JWT_SECRET обязательны: без них Load возвращает ошибку.
func Load() (Config, error) {
	httpPort := 8080
	if v, ok := os.LookupEnv("HTTP_PORT"); ok {
		var err error
		httpPort, err = strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("HTTP_PORT: %w", err)
		}
	}
	if httpPort < 1 || httpPort > 65535 {
		return Config{}, fmt.Errorf("HTTP_PORT: must be between 1 and 65535, got %d", httpPort)
	}

	var databaseURI string
	if v, ok := os.LookupEnv("DATABASE_URI"); ok {
		databaseURI = v
	}

	logLevel := slog.LevelInfo
	if v, ok := os.LookupEnv("LOG_LEVEL"); ok {
		if err := logLevel.UnmarshalText([]byte(v)); err != nil {
			return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
		}
	}

	if databaseURI == "" {
		return Config{}, ErrDatabaseURINotFound
	}

	logFormat := LogFormatJSON
	if v, ok := os.LookupEnv("LOG_FORMAT"); ok {
		logFormat = v
	}
	if logFormat != LogFormatJSON && logFormat != LogFormatText {
		return Config{}, fmt.Errorf("LOG_FORMAT: must be %s or %s, got %q", LogFormatJSON, LogFormatText, logFormat)
	}

	var jwtSecret string
	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		jwtSecret = v
	}
	if jwtSecret == "" {
		return Config{}, ErrJWTSecretNotFound
	}
	if len(jwtSecret) < minJWTSecretLength {
		return Config{}, fmt.Errorf("JWT_SECRET: must be at least %d bytes", minJWTSecretLength)
	}

	var cookieSecure bool
	if v, ok := os.LookupEnv("COOKIE_SECURE"); ok {
		var err error
		cookieSecure, err = strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("COOKIE_SECURE: %w", err)
		}
	}

	return Config{
		HTTPPort:     httpPort,
		DatabaseURI:  databaseURI,
		LogLevel:     logLevel,
		LogFormat:    logFormat,
		JWTSecret:    jwtSecret,
		CookieSecure: cookieSecure,
	}, nil
}
