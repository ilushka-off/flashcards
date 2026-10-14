package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ilushka-off/flashcards/internal/domain"
)

type DeckService interface {
	Create(ctx context.Context, in domain.DeckCreate) (domain.Deck, error)
}

type DeckHandler struct {
	decks  DeckService
	logger *slog.Logger
}

func NewDeckHandler(decks DeckService, logger *slog.Logger) *DeckHandler {
	return &DeckHandler{
		decks:  decks,
		logger: logger,
	}
}

func (h *DeckHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.DeckCreate

	if err := decodeJSON(w, r, &in); err != nil {
		writeServiceError(w, r, err, "")
		h.logger.Error("could not decode request body", err)
		return
	}

	deck, err := h.decks.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, r, err, "")
		h.logger.Error("failed to create deck", err)
		return
	}
	WriteJSON(w, http.StatusCreated, deck)
}
