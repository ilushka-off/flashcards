package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// DeckRepository — то, что сервису нужно от хранилища колод.
type DeckRepository interface {
	CreateDeck(ctx context.Context, in domain.DeckCreate, userID int64) (domain.Deck, error)
	GetDeckByID(ctx context.Context, deckID, userID int64) (domain.Deck, error)
	GetDeckList(ctx context.Context, userID int64) ([]domain.Deck, error)
	UpdateDeckByID(ctx context.Context, deckID, userID int64, in domain.DeckUpdate) (domain.Deck, error)
	DeleteDeckByID(ctx context.Context, deckID, userID int64) error
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

func (s *DeckService) Create(ctx context.Context, in domain.DeckCreate, userID int64) (domain.Deck, error) {
	in.Name = strings.TrimSpace(in.Name)

	if in.Name == "" {
		return domain.Deck{}, &domain.ValidationError{
			Field: "name", Message: "обязательное поле",
		}
	}
	if utf8.RuneCountInString(in.Name) > maxDeckNameLen {
		return domain.Deck{}, &domain.ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("не длиннее %d символов", maxDeckNameLen),
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
			Message: fmt.Sprintf("не длиннее %d символов", maxDeckDescriptionLen),
		}
	}

	deck, err := s.decks.CreateDeck(ctx, in, userID)
	if err != nil {
		return domain.Deck{}, err
	}
	return deck, nil
}

func (s *DeckService) GetByID(ctx context.Context, deckID, userID int64) (domain.Deck, error) {
	deck, err := s.decks.GetDeckByID(ctx, deckID, userID)
	if err != nil {
		return domain.Deck{}, err
	}
	return deck, nil
}

func (s *DeckService) GetList(ctx context.Context, userID int64) ([]domain.Deck, error) {
	decks, err := s.decks.GetDeckList(ctx, userID)
	if err != nil {
		return nil, err
	}
	if decks == nil {
		decks = []domain.Deck{}
	}
	return decks, nil
}

func (s *DeckService) UpdateByID(ctx context.Context, deckID, userID int64, in domain.DeckUpdate) (domain.Deck, error) {
	if in.Name == nil && in.Description == nil {
		return domain.Deck{}, &domain.ValidationError{
			Field:   "body",
			Message: "нужно передать хотя бы одно поле: name, description",
		}
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return domain.Deck{}, &domain.ValidationError{
				Field:   "name",
				Message: "не может быть пустым",
			}
		}
		if utf8.RuneCountInString(name) > maxDeckNameLen {
			return domain.Deck{}, &domain.ValidationError{
				Field:   "name",
				Message: fmt.Sprintf("не длиннее %d символов", maxDeckNameLen),
			}
		}
		in.Name = &name
	}

	if in.Description != nil {
		description := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(description) > maxDeckDescriptionLen {
			return domain.Deck{}, &domain.ValidationError{
				Field:   "description",
				Message: fmt.Sprintf("не длиннее %d символов", maxDeckDescriptionLen),
			}
		}
		in.Description = &description
	}

	deck, err := s.decks.UpdateDeckByID(ctx, deckID, userID, in)
	if err != nil {
		return domain.Deck{}, err
	}
	return deck, nil
}

func (s *DeckService) DeleteByID(ctx context.Context, deckID, userID int64) error {
	return s.decks.DeleteDeckByID(ctx, deckID, userID)
}
