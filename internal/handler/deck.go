package handler

import (
	"context"
	"net/http"

	"github.com/ilushka-off/flashcards/internal/domain"
)

type DeckService interface {
	Create(ctx context.Context, in domain.DeckCreate) (domain.Deck, error)
}

type DeckHandler struct {
	decks DeckService
}

func NewDeckHandler(decks DeckService) *DeckHandler {
	return &DeckHandler{
		decks: decks,
	}
}

func (h *DeckHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in domain.DeckCreate

	if err := decodeJSON(r, &in); err != nil {
		writeServiceError(w, r, err, "")
		return
	}

	deck, err := h.decks.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, r, err, "")
		return
	}
	WriteJSON(w, http.StatusCreated, deck)
}
