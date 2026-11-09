package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// CardRepository — то, что сервису нужно от хранилища карточек.
type CardRepository interface {
	CreateCard(ctx context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error)
	// GetCardList возвращает страницу карточек и общее число подходящих под q.
	// q уже обрезан; экранировать спецсимволы LIKE — забота repository.
	GetCardList(ctx context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error)
}

const (
	maxCardSideLen    = 200
	maxCardExampleLen = 500
	maxCardListLimit  = 100
	maxQuerySideLen   = 100
)

type CardService struct {
	cards CardRepository
}

func NewCardService(cards CardRepository) *CardService {
	return &CardService{
		cards: cards,
	}
}

// Create добавляет карточку в колоду пользователя. Начальное расписание SM-2
// (EF 2.5, due_at = сейчас) задаёт база, поэтому карточка сразу попадает
// в очередь повторения.
func (s *CardService) Create(ctx context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error) {
	front, err := validateCardSide("front", in.Front)
	if err != nil {
		return domain.Card{}, err
	}
	back, err := validateCardSide("back", in.Back)
	if err != nil {
		return domain.Card{}, err
	}

	example := strings.TrimSpace(in.Example)
	if utf8.RuneCountInString(example) > maxCardExampleLen {
		return domain.Card{}, &domain.ValidationError{
			Field:   "example",
			Message: fmt.Sprintf("не длиннее %d символов", maxCardExampleLen),
		}
	}

	card, err := s.cards.CreateCard(ctx, deckID, userID, domain.CardCreate{
		Front:   front,
		Back:    back,
		Example: example,
	})
	if err != nil {
		return domain.Card{}, err
	}
	return card, nil
}

// GetList возвращает страницу карточек колоды пользователя, новые сверху.
// q ищет подстроку в front или back без учёта регистра; пустой q — без фильтра.
// Чужая или несуществующая колода — ErrNotFound из repository: пустой список
// и отсутствие колоды клиент должен различать.
// Второе значение — total: сколько карточек подходит под фильтр всего,
// а не на текущей странице.
func (s *CardService) GetList(ctx context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error) {
	if limit < 1 || limit > maxCardListLimit {
		return nil, 0, &domain.ValidationError{
			Field:   "limit",
			Message: fmt.Sprintf("целое число от 1 до %d", maxCardListLimit),
		}
	}
	if offset < 0 {
		return nil, 0, &domain.ValidationError{
			Field: "offset", Message: "целое число не меньше 0",
		}
	}

	query := strings.TrimSpace(q)
	if utf8.RuneCountInString(query) > maxQuerySideLen {
		return nil, 0, &domain.ValidationError{
			Field:   "q",
			Message: fmt.Sprintf("не длиннее %d символов", maxQuerySideLen),
		}
	}

	cards, total, err := s.cards.GetCardList(ctx, deckID, userID, limit, offset, query)
	if err != nil {
		return nil, 0, err
	}
	return cards, total, nil
}

// validateCardSide проверяет front или back: обязательное поле не длиннее
// maxCardSideLen символов. Возвращает значение без пробелов по краям.
func validateCardSide(field, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", &domain.ValidationError{
			Field: field, Message: "обязательное поле",
		}
	}
	if utf8.RuneCountInString(value) > maxCardSideLen {
		return "", &domain.ValidationError{
			Field:   field,
			Message: fmt.Sprintf("не длиннее %d символов", maxCardSideLen),
		}
	}
	return value, nil
}
