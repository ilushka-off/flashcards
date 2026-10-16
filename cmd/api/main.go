package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/ilushka-off/flashcards/internal/config"
	"github.com/ilushka-off/flashcards/internal/handler"
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	err = postgres.RunMigrations(cfg.DatabaseURI)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

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

	slog.Info("server started", "port", cfg.HTTPPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), router); err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
