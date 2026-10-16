package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// DeckRepository — то, что сервису нужно от хранилища колод.
type DeckRepository interface {
	CreateDeck(ctx context.Context, in domain.DeckCreate) (domain.Deck, error)
	GetDeckByID(ctx context.Context, deckID int64) (domain.Deck, error)
	GetDeckList(ctx context.Context) ([]domain.Deck, error)
}

const (
	maxDeckNameLen        = 100
	maxDeckDescriptionLen = 500
)

type DeckService struct {
	decks DeckRepository
}

func NewDeckService(decks DeckRepository) *DeckService {
	return &DeckService{
		decks: decks,
	}
}

func (s *DeckService) Create(ctx context.Context, in domain.DeckCreate) (domain.Deck, error) {
	in.Name = strings.TrimSpace(in.Name)

	if in.Name == "" {
		return domain.Deck{}, &domain.ValidationError{
			Field: "name", Message: "обязательное поле",
		}
	}
	if utf8.RuneCountInString(in.Name) > maxDeckNameLen {
		return domain.Deck{}, &domain.ValidationError{
			Field:   "name",
			Message: "не длиннее 100 символов",
		}
	}

	if in.Language != domain.LanguageEN && in.Language != domain.LanguageES {
		return domain.Deck{}, &domain.ValidationError{
			Field:   "language",
			Message: "допустимые значения en, es",
		}
	}

	in.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(in.Description) > maxDeckDescriptionLen {
		return domain.Deck{}, &domain.ValidationError{
			Field:   "description",
			Message: "не длиннее 500 символов",
		}
	}

	deck, err := s.decks.CreateDeck(ctx, in)
	if err != nil {
		return domain.Deck{}, err
	}
	return deck, nil
}

func (s *DeckService) GetByID(ctx context.Context, deckID int64) (domain.Deck, error) {
	deck, err := s.decks.GetDeckByID(ctx, deckID)
	if err != nil {
		return domain.Deck{}, err
	}
	return deck, nil
}

func (s *DeckService) GetList(ctx context.Context) ([]domain.Deck, error) {
	decks, err := s.decks.GetDeckList(ctx)
	if err != nil {
		return nil, err
	}
	if decks == nil {
		decks = []domain.Deck{}
	}
	return decks, nil
}
