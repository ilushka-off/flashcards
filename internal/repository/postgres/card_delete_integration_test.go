//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestCardRepositoryDeleteByID(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	decks := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")
	deck, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	// Deleting card 2 from deck 1 also checks that the query uses the card ID.
	seeded := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{
		{Front: "uno", Back: "one"},
		{Front: "dos", Back: "two"},
	})
	kept, deleted := seeded[0], seeded[1]
	for _, card := range seeded {
		_, err := pool.Exec(ctx, `INSERT INTO reviews (card_id, grade, reviewed_at, interval_days, ease_factor)
			VALUES ($1, 'good', now(), 1, 2.5), ($1, 'easy', now(), 4, 2.6)`, card.ID)
		if err != nil {
			t.Fatalf("seed reviews: %v", err)
		}
	}

	if err := cards.DeleteByID(ctx, deleted.ID, ownerID); err != nil {
		t.Fatalf("DeleteByID() error = %v", err)
	}
	var cardCount, reviewCount int
	err = pool.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM cards WHERE id = $1),
		(SELECT COUNT(*) FROM reviews WHERE card_id = $1)`, deleted.ID).Scan(&cardCount, &reviewCount)
	if err != nil {
		t.Fatalf("read deleted card and reviews: %v", err)
	}
	if cardCount != 0 || reviewCount != 0 {
		t.Errorf("deleted card has (cards %d, reviews %d), want (0, 0)", cardCount, reviewCount)
	}
	assertStoredCard(t, cards, ownerID, kept)
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE card_id = $1`, kept.ID).Scan(&reviewCount); err != nil {
		t.Fatalf("read retained reviews: %v", err)
	}
	if reviewCount != 2 {
		t.Errorf("other card reviews = %d, want 2", reviewCount)
	}
	remaining, err := decks.GetDeckByID(ctx, deck.ID, ownerID)
	if err != nil {
		t.Fatalf("GetDeckByID() after deleting card: %v", err)
	}
	if remaining.TotalCards != 1 || remaining.DueCards != 1 {
		t.Errorf("deck counters = (total %d, due %d), want (1, 1)", remaining.TotalCards, remaining.DueCards)
	}
	if err := cards.DeleteByID(ctx, deleted.ID, ownerID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("repeated DeleteByID() error = %v, want ErrNotFound", err)
	}
}

func TestCardRepositoryDeleteByIDErrors(t *testing.T) {
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
	// Owning another deck must not grant access to the owner's card.
	if _, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "English", Language: domain.LanguageEN}, strangerID); err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	card := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{{Front: "uno", Back: "one"}})[0]
	_, err = pool.Exec(ctx, `INSERT INTO reviews (card_id, grade, reviewed_at, interval_days, ease_factor)
		VALUES ($1, 'good', now(), 1, 2.5)`, card.ID)
	if err != nil {
		t.Fatalf("seed review: %v", err)
	}
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	tests := []struct {
		name    string
		ctx     context.Context
		cardID  int64
		userID  int64
		wantErr error
	}{
		{name: "foreign card", ctx: ctx, cardID: card.ID, userID: strangerID, wantErr: domain.ErrNotFound},
		{name: "missing card", ctx: ctx, cardID: card.ID + 100, userID: ownerID, wantErr: domain.ErrNotFound},
		{name: "missing user", ctx: ctx, cardID: card.ID, userID: strangerID + 100, wantErr: domain.ErrNotFound},
		{name: "canceled context", ctx: canceledCtx, cardID: card.ID, userID: ownerID, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := cards.DeleteByID(tt.ctx, tt.cardID, tt.userID); !errors.Is(err, tt.wantErr) {
				t.Errorf("DeleteByID() error = %v, want %v", err, tt.wantErr)
			}
			assertStoredCard(t, cards, ownerID, card)
			var reviewCount int
			if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE card_id = $1`, card.ID).Scan(&reviewCount); err != nil {
				t.Fatalf("read retained review: %v", err)
			}
			if reviewCount != 1 {
				t.Errorf("reviews after failed deletion = %d, want 1", reviewCount)
			}
		})
	}
}
