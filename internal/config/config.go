package config

import (
	"errors"
	"flag"
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

func Load() (Config, error) {
	var cfg Config

	var httpPort int
	if v, ok := os.LookupEnv("HTTP_PORT"); ok {

		var err error

		httpPort, err = strconv.Atoi(v)
		if err != nil {
			return cfg, fmt.Errorf("invalid port: %w", err)
		}
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

	flag.IntVar(&cfg.HTTPPort, "port", httpPort, "Server port")
	flag.StringVar(&cfg.DatabaseURI, "db", databaseURI, "Database URI connection")
	flag.TextVar(&cfg.LogLevel, "log", logLevel, "log level: debug, info, warn, error")
	flag.Parse()

	if cfg.DatabaseURI == "" {
		return Config{}, ErrDatabaseURINotFound
	}

	return cfg, nil
}
