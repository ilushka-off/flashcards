package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ilushka-off/flashcards/internal/domain"
)

type CardService interface {
	Create(ctx context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error)
}

type CardHandler struct {
	cards  CardService
	logger *slog.Logger
}

func NewCardHandler(cards CardService, logger *slog.Logger) *CardHandler {
	return &CardHandler{
		cards:  cards,
		logger: logger,
	}
}

// Create обрабатывает POST /decks/{id}/cards.
func (h *CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || deckID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректный id в path запросе")
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	var in domain.CardCreate
	if err := decodeJSON(w, r, &in); err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}

	card, err := h.cards.Create(r.Context(), deckID, userID, in)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "колода не найдена")
		return
	}
	WriteJSON(w, http.StatusCreated, card)
}
