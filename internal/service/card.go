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
}

const (
	maxCardSideLen    = 200
	maxCardExampleLen = 500
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
