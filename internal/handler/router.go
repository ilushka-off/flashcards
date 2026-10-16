package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ilushka-off/flashcards/internal/middleware"
)

func NewRouter(decksHandler *DeckHandler, healthHandler *HealthHandler, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger(logger))

	r.MethodNotAllowed(methodNotAllowed)
	r.NotFound(notFound)

	r.Get("/health", healthHandler.Check)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/decks", func(r chi.Router) {
			r.Get("/", decksHandler.GetList)
			r.Get("/{id}", decksHandler.GetByID)
			r.Post("/", decksHandler.Create)
		})
	})

	return r
}
