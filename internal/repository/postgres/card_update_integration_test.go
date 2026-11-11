//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCardRepositoryUpdateCardByID(t *testing.T) {
	pool := newTestPool(t)
	cards := NewCardRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")
	deck, err := NewDeckRepository(pool).CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	// A second card must not change, and card IDs must differ from the deck ID.
	untouched := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{{Front: "dos", Back: "two"}})[0]

	front, back, example, empty := "hola", "hello", "Hola, mundo!", ""
	tests := []struct {
		name string
		in   domain.CardUpdate
		want domain.CardCreate
	}{
		{name: "front only", in: domain.CardUpdate{Front: &front}, want: domain.CardCreate{Front: front, Back: "one", Example: "uno, dos"}},
		{name: "back only", in: domain.CardUpdate{Back: &back}, want: domain.CardCreate{Front: "uno", Back: back, Example: "uno, dos"}},
		{name: "example only", in: domain.CardUpdate{Example: &example}, want: domain.CardCreate{Front: "uno", Back: "one", Example: example}},
		{name: "clear example", in: domain.CardUpdate{Example: &empty}, want: domain.CardCreate{Front: "uno", Back: "one", Example: ""}},
		{name: "all text fields", in: domain.CardUpdate{Front: &front, Back: &back, Example: &example}, want: domain.CardCreate{Front: front, Back: back, Example: example}},
	}

	// Deliberately use a supplied time with a non-UTC zone and microsecond precision.
	now := time.Date(2026, 11, 10, 12, 30, 0, 123456000, time.FixedZone("MSK", 3*60*60))
	dueAt := now.Add(14 * 24 * time.Hour).UTC()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{{Front: "uno", Back: "one", Example: "uno, dos"}})[0]
			_, err := pool.Exec(ctx, `UPDATE cards SET ease_factor = 2.15, interval_days = 14,
				repetitions = 4, due_at = $2, updated_at = $3 WHERE id = $1`, original.ID, dueAt, now.Add(-time.Hour))
			if err != nil {
				t.Fatalf("seed schedule: %v", err)
			}
			_, err = pool.Exec(ctx, `INSERT INTO reviews (card_id, grade, reviewed_at, interval_days, ease_factor)
				VALUES ($1, 'good', $2, 14, 2.15)`, original.ID, now.Add(-time.Hour))
			if err != nil {
				t.Fatalf("seed review: %v", err)
			}

			want := original
			want.Front, want.Back, want.Example = tt.want.Front, tt.want.Back, tt.want.Example
			want.EaseFactor, want.IntervalDays, want.Repetitions = 2.15, 14, 4
			want.DueAt, want.UpdatedAt = dueAt, now.UTC()
			got, err := cards.UpdateCardByID(ctx, original.ID, ownerID, tt.in, now)
			if err != nil {
				t.Fatalf("UpdateCardByID() error = %v", err)
			}
			if got != want {
				t.Errorf("card = %+v, want %+v", got, want)
			}
			for name, value := range map[string]time.Time{"due_at": got.DueAt, "created_at": got.CreatedAt, "updated_at": got.UpdatedAt} {
				if value.Location() != time.UTC {
					t.Errorf("%s location = %v, want UTC", name, value.Location())
				}
			}
			assertStoredCard(t, cards, ownerID, want)

			var count int
			err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE card_id = $1
				AND grade = 'good' AND reviewed_at = $2 AND interval_days = 14 AND ease_factor = 2.15`,
				original.ID, now.Add(-time.Hour)).Scan(&count)
			if err != nil {
				t.Fatalf("read review: %v", err)
			}
			if count != 1 {
				t.Errorf("unchanged reviews = %d, want 1", count)
			}

			// Repeating a patch keeps the same text and schedule but advances updated_at.
			later := now.Add(time.Minute)
			repeated, err := cards.UpdateCardByID(ctx, original.ID, ownerID, tt.in, later)
			if err != nil {
				t.Fatalf("repeat UpdateCardByID() error = %v", err)
			}
			want.UpdatedAt = later.UTC()
			if repeated != want {
				t.Errorf("repeated card = %+v, want %+v", repeated, want)
			}
		})
	}
	assertStoredCard(t, cards, ownerID, untouched)
}

func TestCardRepositoryUpdateCardByIDErrors(t *testing.T) {
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
	// The stranger owns a deck too: joining any owned deck must not grant access.
	if _, err := decks.CreateDeck(ctx, domain.DeckCreate{Name: "Other", Language: domain.LanguageEN}, strangerID); err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	original := seedCards(t, cards, deck.ID, ownerID, []domain.CardCreate{{Front: "uno", Back: "one", Example: "uno, dos"}})[0]
	front, back, empty := "changed", "new translation", ""
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	tests := []struct {
		name    string
		ctx     context.Context
		cardID  int64
		userID  int64
		in      domain.CardUpdate
		wantErr error
	}{
		{name: "missing card", ctx: ctx, cardID: original.ID + 100, userID: ownerID, in: domain.CardUpdate{Front: &front}, wantErr: domain.ErrNotFound},
		{name: "another user's card", ctx: ctx, cardID: original.ID, userID: strangerID, in: domain.CardUpdate{Front: &front}, wantErr: domain.ErrNotFound},
		{name: "missing user", ctx: ctx, cardID: original.ID, userID: strangerID + 100, in: domain.CardUpdate{Front: &front}, wantErr: domain.ErrNotFound},
		{name: "canceled context", ctx: canceledCtx, cardID: original.ID, userID: ownerID, in: domain.CardUpdate{Front: &front}, wantErr: context.Canceled},
		{name: "database constraint", ctx: ctx, cardID: original.ID, userID: ownerID, in: domain.CardUpdate{Front: &empty, Back: &back}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cards.UpdateCardByID(tt.ctx, tt.cardID, tt.userID, tt.in, now)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("UpdateCardByID() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) {
					t.Errorf("UpdateCardByID() error = %v, want wrapped PostgreSQL error", err)
				}
			}
			if got != (domain.Card{}) {
				t.Errorf("card on error = %+v, want zero value", got)
			}
			assertStoredCard(t, cards, ownerID, original)
		})
	}
}

// Read back through the list query to verify the persisted row independently.
func assertStoredCard(t *testing.T, cards *CardRepository, userID int64, want domain.Card) {
	t.Helper()
	stored, _, err := cards.GetCardList(context.Background(), want.DeckID, userID, 100, 0, "")
	if err != nil {
		t.Fatalf("GetCardList() error = %v", err)
	}
	for _, card := range stored {
		if card.ID == want.ID {
			if card != want {
				t.Errorf("stored card = %+v, want %+v", card, want)
			}
			return
		}
	}
	t.Fatalf("card %d missing from deck %d", want.ID, want.DeckID)
}
