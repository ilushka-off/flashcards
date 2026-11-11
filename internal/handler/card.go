package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ilushka-off/flashcards/internal/domain"
)

const (
	defaultLimit  = 20
	defaultOffset = 0
)

type CardService interface {
	Create(ctx context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error)
	GetList(ctx context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error)
	UpdateByID(ctx context.Context, cardID, userID int64, in domain.CardUpdate) (domain.Card, error)
	DeleteByID(ctx context.Context, cardID, userID int64) error
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
		WriteError(w, http.StatusBadRequest, CodeValidation, "id: должен быть положительным целым числом")
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

// GetList обрабатывает GET /decks/{id}/cards.
func (h *CardHandler) GetList(w http.ResponseWriter, r *http.Request) {
	deckID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || deckID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "id: должен быть положительным целым числом")
		return
	}

	limit, ok := queryInt(r, "limit", defaultLimit)
	if !ok {
		WriteError(w, http.StatusBadRequest, CodeValidation, "limit: должен быть целым числом")
		return
	}

	offset, ok := queryInt(r, "offset", defaultOffset)
	if !ok {
		WriteError(w, http.StatusBadRequest, CodeValidation, "offset: должен быть целым числом")
		return
	}
	q := r.URL.Query().Get("q")

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	cards, total, err := h.cards.GetList(r.Context(), deckID, userID, limit, offset, q)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "колода не найдена")
		return
	}
	WriteJSON(w, http.StatusOK, Items[domain.Card]{Items: cards, Total: &total})
}

// UpdateByID обрабатывает PATCH /cards/{id}.
func (h *CardHandler) UpdateByID(w http.ResponseWriter, r *http.Request) {
	cardID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || cardID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "id: должен быть положительным целым числом")
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	var in domain.CardUpdate
	if err := decodeJSON(w, r, &in); err != nil {
		writeServiceError(w, r, h.logger, err, "")
		return
	}

	card, err := h.cards.UpdateByID(r.Context(), cardID, userID, in)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "карточка не найдена")
		return
	}
	WriteJSON(w, http.StatusOK, card)

}

// DeleteByID обрабатывает DELETE /cards/{id}.
func (h *CardHandler) DeleteByID(w http.ResponseWriter, r *http.Request) {
	cardID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || cardID < 1 {
		WriteError(w, http.StatusBadRequest, CodeValidation, "id: должен быть положительным целым числом")
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	err = h.cards.DeleteByID(r.Context(), cardID, userID)
	if err != nil {
		writeServiceError(w, r, h.logger, err, "карточка не найдена")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// queryInt читает целочисленный query-параметр key.
// Если параметр не задан, возвращает defaultValue и true.
// Если значение не целое число, возвращает false. Диапазон проверяет сервис.
func queryInt(r *http.Request, key string, defaultValue int) (int, bool) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultValue, true
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}
