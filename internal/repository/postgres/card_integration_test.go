//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestCardRepositoryCreateCard(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	decks := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	deck, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	in := domain.CardCreate{Front: "uno", Back: "one", Example: "uno, dos, tres"}
	before := time.Now()
	got, err := cards.CreateCard(ctx, deck.ID, ownerID, in)
	if err != nil {
		t.Fatalf("CreateCard() error = %v", err)
	}

	if got.ID < 1 {
		t.Errorf("ID = %d, want positive", got.ID)
	}
	if got.DeckID != deck.ID || got.Front != in.Front || got.Back != in.Back || got.Example != in.Example {
		t.Errorf("card = %+v, want deck %d with %+v", got, deck.ID, in)
	}

	// Initial SM-2 schedule comes from the schema defaults.
	if got.EaseFactor != 2.5 || got.IntervalDays != 0 || got.Repetitions != 0 {
		t.Errorf("schedule = (ease %v, interval %d, reps %d), want (2.5, 0, 0)", got.EaseFactor, got.IntervalDays, got.Repetitions)
	}
	// now() is fixed within a transaction, so all three timestamps match exactly.
	if !got.DueAt.Equal(got.CreatedAt) || !got.UpdatedAt.Equal(got.CreatedAt) {
		t.Errorf("due_at %v, created_at %v, updated_at %v, want all equal", got.DueAt, got.CreatedAt, got.UpdatedAt)
	}
	if d := got.CreatedAt.Sub(before); d < -time.Minute || d > time.Minute {
		t.Errorf("created_at = %v, want close to %v", got.CreatedAt, before)
	}
	if got.CreatedAt.Location() != time.UTC {
		t.Errorf("created_at location = %v, want UTC", got.CreatedAt.Location())
	}

	// The new card is due right away, so it counts in due_cards.
	after, err := decks.GetDeckByID(ctx, deck.ID, ownerID)
	if err != nil {
		t.Fatalf("GetDeckByID() error = %v", err)
	}
	if after.TotalCards != 1 || after.DueCards != 1 {
		t.Errorf("counters = (total %d, due %d), want (1, 1)", after.TotalCards, after.DueCards)
	}
}

func TestCardRepositoryCreateCardNotFound(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	decks := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")
	strangerID := createTestUser(t, pool, "stranger@mail.ru")

	deck, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	tests := []struct {
		name   string
		deckID int64
		userID int64
	}{
		{name: "deck does not exist", deckID: deck.ID + 100, userID: ownerID},
		{name: "deck belongs to another user", deckID: deck.ID, userID: strangerID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cards.CreateCard(ctx, tt.deckID, tt.userID, domain.CardCreate{Front: "uno", Back: "one"})
			if !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("CreateCard() error = %v, want ErrNotFound", err)
			}
		})
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM cards`).Scan(&count); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if count != 0 {
		t.Errorf("cards in table = %d, want 0", count)
	}
}
