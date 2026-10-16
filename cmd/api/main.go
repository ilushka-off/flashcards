package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/ilushka-off/flashcards/internal/config"
	"github.com/ilushka-off/flashcards/internal/handler"
	"github.com/ilushka-off/flashcards/internal/middleware"
	"github.com/ilushka-off/flashcards/internal/repository/postgres"
	"github.com/ilushka-off/flashcards/internal/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := newLogger(cfg)
	// SetDefault — только запасной вариант для редких вызовов без переданного логгера.
	slog.SetDefault(logger)

	err = postgres.RunMigrations(cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	logger.Info("migrations applied")

	pool, err := postgres.OpenPool(context.Background(), cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("open pool: %w", err)
	}
	defer pool.Close()

	healthHandler := handler.NewHealthHandler(pool, logger)

	deckRepository := postgres.NewDeckRepository(pool)
	deckService := service.NewDeckService(deckRepository)
	deckHandler := handler.NewDeckHandler(deckService, logger)

	router := handler.NewRouter(deckHandler, healthHandler, logger)

	logger.Info("server started", "port", cfg.HTTPPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), router); err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// newLogger строит логгер по конфигу: text для локального чтения, json для docker,
// и оборачивает его ContextHandler, чтобы в каждую запись попадал request_id.
func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}

	var base slog.Handler
	if cfg.LogFormat == config.LogFormatText {
		base = slog.NewTextHandler(os.Stdout, opts)
	} else {
		base = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(middleware.NewContextHandler(base))
}
