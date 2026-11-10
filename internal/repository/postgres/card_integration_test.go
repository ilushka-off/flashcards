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

// seedCards creates the cards in order, so the last one is the newest.
func seedCards(t *testing.T, cards *CardRepository, deckID, userID int64, in []domain.CardCreate) []domain.Card {
	t.Helper()
	out := make([]domain.Card, 0, len(in))
	for _, c := range in {
		card, err := cards.CreateCard(context.Background(), deckID, userID, c)
		if err != nil {
			t.Fatalf("CreateCard(%+v) error = %v", c, err)
		}
		out = append(out, card)
	}
	return out
}

func cardIDs(cards []domain.Card) []int64 {
	ids := make([]int64, 0, len(cards))
	for _, c := range cards {
		ids = append(ids, c.ID)
	}
	return ids
}

func equalIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCardRepositoryGetCardListPagination(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	decks := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	deck, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	seeded := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{
		{Front: "uno", Back: "one"}, {Front: "dos", Back: "two"}, {Front: "tres", Back: "three"},
		{Front: "cuatro", Back: "four"}, {Front: "cinco", Back: "five"},
	})
	// Newest first: cinco, cuatro, tres, dos, uno.
	newest := []int64{seeded[4].ID, seeded[3].ID, seeded[2].ID, seeded[1].ID, seeded[0].ID}

	// A card in another deck must not leak into this deck's list or total.
	other, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Other", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	seedCards(t, cards, other.ID, ownerID, []domain.CardCreate{{Front: "seis", Back: "six"}})

	tests := []struct {
		name    string
		limit   int
		offset  int
		wantIDs []int64
	}{
		{name: "first page", limit: 2, offset: 0, wantIDs: newest[0:2]},
		{name: "second page", limit: 2, offset: 2, wantIDs: newest[2:4]},
		{name: "last partial page", limit: 2, offset: 4, wantIDs: newest[4:5]},
		{name: "all at once", limit: 100, offset: 0, wantIDs: newest},
		// Past the end the page is empty, but total still counts every card.
		{name: "offset past the end", limit: 2, offset: 10, wantIDs: []int64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := cards.GetCardList(ctx, deck.ID, ownerID, tt.limit, tt.offset, "")
			if err != nil {
				t.Fatalf("GetCardList() error = %v", err)
			}
			if total != 5 {
				t.Errorf("total = %d, want 5", total)
			}
			if ids := cardIDs(got); !equalIDs(ids, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", ids, tt.wantIDs)
			}
			if got == nil {
				t.Error("cards = nil, want empty slice so JSON gets [] not null")
			}
		})
	}
}

func TestCardRepositoryGetCardListSearch(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	decks := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	deck, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "English", Language: domain.LanguageEN}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	seeded := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{
		{Front: "Deadline", Back: "крайний срок"}, // 0
		{Front: "meeting", Back: "встреча"},       // 1
		{Front: "100%", Back: "полностью"},        // 2
		{Front: "1000", Back: "тысяча"},           // 3
		{Front: "snake_case", Back: "змеиный"},    // 4
		{Front: "snakeXcase", Back: "не то"},      // 5
		{Front: `back\slash`, Back: "слэш"},       // 6
	})
	id := func(i int) int64 { return seeded[i].ID }

	tests := []struct {
		name    string
		q       string
		wantIDs []int64
	}{
		{name: "front, case insensitive", q: "dead", wantIDs: []int64{id(0)}},
		{name: "front, upper case query", q: "MEET", wantIDs: []int64{id(1)}},
		{name: "back, cyrillic case insensitive", q: "КРАЙНИЙ", wantIDs: []int64{id(0)}},
		{name: "substring in the middle", q: "eti", wantIDs: []int64{id(1)}},
		// Without escaping, % would match every card and _ any single character.
		{name: "percent is literal", q: "%", wantIDs: []int64{id(2)}},
		{name: "underscore is literal", q: "_", wantIDs: []int64{id(4)}},
		{name: "underscore inside word is literal", q: "snake_", wantIDs: []int64{id(4)}},
		{name: "backslash is literal", q: `\`, wantIDs: []int64{id(6)}},
		{name: "nothing found", q: "zzz", wantIDs: []int64{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := cards.GetCardList(ctx, deck.ID, ownerID, 100, 0, tt.q)
			if err != nil {
				t.Fatalf("GetCardList() error = %v", err)
			}
			if ids := cardIDs(got); !equalIDs(ids, tt.wantIDs) {
				t.Errorf("ids = %v, want %v", ids, tt.wantIDs)
			}
			// total is filtered by the same q as the page.
			if total != len(tt.wantIDs) {
				t.Errorf("total = %d, want %d", total, len(tt.wantIDs))
			}
		})
	}
}

func TestCardRepositoryGetCardListNotFound(t *testing.T) {
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
	seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{{Front: "uno", Back: "one"}})

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
			got, total, err := cards.GetCardList(ctx, tt.deckID, tt.userID, 20, 0, "")
			if !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("GetCardList() error = %v, want ErrNotFound", err)
			}
			if len(got) != 0 || total != 0 {
				t.Errorf("GetCardList() = (%d cards, total %d), want nothing", len(got), total)
			}
		})
	}
}

// An existing deck with no cards is not "not found": the client gets 200 with [].
func TestCardRepositoryGetCardListEmptyDeck(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	decks := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	deck, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Empty", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	got, total, err := cards.GetCardList(ctx, deck.ID, ownerID, 20, 0, "")
	if err != nil {
		t.Fatalf("GetCardList() error = %v, want nil for an empty deck", err)
	}
	if got == nil || len(got) != 0 || total != 0 {
		t.Errorf("GetCardList() = (%v, %d), want ([], 0)", got, total)
	}
}
