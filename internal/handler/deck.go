package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/ilushka-off/flashcards/internal/userctx"
)

type DeckService interface {
	Create(ctx context.Context, in domain.DeckCreate, userID int64) (domain.Deck, error)
	GetByID(ctx context.Context, deckID, userID int64) (domain.Deck, error)
	GetList(ctx context.Context, userID int64) ([]domain.Deck, error)
	UpdateByID(ctx context.Context, deckID, userID int64, in domain.DeckUpdate) (domain.Deck, error)
	DeleteByID(ctx context.Context, deckID, userID int64) error
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

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	if err := decodeJSON(w, r, &in); err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}

	deck, err := h.decks.Create(r.Context(), in, userID)
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

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	deck, err := h.decks.GetByID(r.Context(), deckID, userID)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "колода не найдена")
		return
	}
	WriteJSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) GetList(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	decks, err := h.decks.GetList(r.Context(), userID)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}
	WriteJSON(w, http.StatusOK, Items[domain.Deck]{Items: decks})
}

func (h *DeckHandler) UpdateByID(w http.ResponseWriter, r *http.Request) {
	var in domain.DeckUpdate

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	if err := decodeJSON(w, r, &in); err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}

	deckID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || deckID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректный id в path запросе")
		return
	}

	deck, err := h.decks.UpdateByID(r.Context(), deckID, userID, in)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "колода не найдена")
		return
	}
	WriteJSON(w, http.StatusOK, deck)
}

func (h *DeckHandler) DeleteByID(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || deckID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "некорректный id в path запросе")
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	if err := h.decks.DeleteByID(r.Context(), deckID, userID); err != nil {
		writeServiceError(w, r, h.logger, err, "колода не найдена")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func getUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := userctx.UserIDFrom(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "требуется авторизация")
	}
	return userID, ok
}
