package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/middleware"
)

// NewRouter собирает маршруты API на chi и возвращает готовый http.Handler.
// Цепочка middleware: RequestID → Logging → Recover → mux, затем /health,
// /api/v1/auth/register и группа /api/v1/decks; неизвестные пути — 404 в едином
// формате ошибок API.
func NewRouter(decksHandler *DeckHandler, healthHandler *HealthHandler, userHandler *UserHandler, token *auth.TokenManager, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(Recover(logger))

	r.NotFound(notFound)

	r.Get("/health", healthHandler.Check)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", userHandler.Register)
			r.Post("/login", userHandler.Login)
		})
		r.Group(func(r chi.Router) {
			r.Use(Auth(token))
			r.Get("/auth/me", userHandler.Me)
			r.Route("/decks", func(r chi.Router) {
				r.Get("/", decksHandler.GetList)
				r.Get("/{id}", decksHandler.GetByID)
				r.Post("/", decksHandler.Create)
				r.Patch("/{id}", decksHandler.UpdateByID)
			})
		})
	})

	return r
}
