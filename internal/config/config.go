package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config — параметры запуска сервиса.
type Config struct {
	HTTPPort    int
	DatabaseURI string
	LogLevel    slog.Level
}

// ErrDatabaseURINotFound Ошибка конфигурации, при которых сервис не может быть запущен.
var (
	ErrDatabaseURINotFound = errors.New("database uri not found")
)

// Load собирает конфигурацию из переменных окружения
// HTTP_PORT, DATABASE_URI, LOG_LEVEL.
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

	return Config{
		HTTPPort:    httpPort,
		DatabaseURI: databaseURI,
		LogLevel:    logLevel,
	}, nil
}
