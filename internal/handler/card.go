package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ilushka-off/flashcards/internal/domain"
)

const (
	defaultLimit  = 20
	maxLimit      = 100
	defaultOffset = 0
)

type CardService interface {
	Create(ctx context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error)
	GetList(ctx context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error)
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

	limit, err := parseQueryParams(r, "limit", defaultLimit, 1)
	if err != nil || limit > maxLimit {
		WriteError(w, http.StatusBadRequest, CodeValidation, "limit: целое число от 1 до 100")
		return
	}

	offset, err := parseQueryParams(r, "offset", defaultOffset, 0)
	if err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidation, "offset: целое число не меньше 0")
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

// parseQueryParams читает целочисленный query-параметр key.
// Если параметр не задан, возвращает defaultValue.
// Если значение не число или меньше minValue, возвращает ошибку.
func parseQueryParams(r *http.Request, key string, defaultValue, minValue int) (int, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultValue, nil
	}

	v, err := strconv.Atoi(s)
	if err != nil || v < minValue {
		return 0, fmt.Errorf("%s: целое число не меньше %d", key, minValue)
	}
	return v, nil
}
