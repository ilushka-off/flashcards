package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ilushka-off/flashcards/internal/config"
	"github.com/ilushka-off/flashcards/internal/repository/postgres"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.OpenPool(ctx, cfg.DatabaseURI)
	if err != nil {
		return err
	}
	defer pool.Close()

	err = postgres.Run(cfg.DatabaseURI)
	if err != nil {
		return err
	}
	handler := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "I'm a live")
	}

	http.HandleFunc(`/healthz`, handler)
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.HTTPPort), nil)
	if err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
