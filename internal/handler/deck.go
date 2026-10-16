package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ilushka-off/flashcards/internal/domain"
)

type DeckService interface {
	Create(ctx context.Context, in domain.DeckCreate) (domain.Deck, error)
	GetByID(ctx context.Context, deckID int64) (domain.Deck, error)
	GetList(ctx context.Context) ([]domain.Deck, error)
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
		writeServiceError(w, r, h.logger, err, "")
		return
	}

	deck, err := h.decks.Create(r.Context(), in)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}
	WriteJSON(w, http.StatusCreated, deck)
}

func (h *DeckHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || deckID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректный id в path запросе")
		return
	}

	deck, err := h.decks.GetByID(r.Context(), deckID)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "колода не найдена")
		return
	}
	WriteJSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) GetList(w http.ResponseWriter, r *http.Request) {
	decks, err := h.decks.GetList(r.Context())
	if err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}
	WriteJSON(w, http.StatusOK, Items[domain.Deck]{Items: decks})
}
