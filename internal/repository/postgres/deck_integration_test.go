//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestPool applies migrations, opens a pool against TEST_DATABASE_URI and
// truncates all tables before and after the test. It skips when the variable
// is unset so `go test -tags integration ./...` stays green without a database.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	uri := os.Getenv("TEST_DATABASE_URI")
	if uri == "" {
		t.Skip("TEST_DATABASE_URI is not set, skipping integration test")
	}

	if err := RunMigrations(uri); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := OpenPool(context.Background(), uri)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	truncateAll(t, pool)
	t.Cleanup(func() { truncateAll(t, pool) })
	return pool
}

func truncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `TRUNCATE refresh_tokens, reviews, cards, decks, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

// createTestUser inserts a deck owner, since decks.user_id is a NOT NULL foreign key.
func createTestUser(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	user, err := NewUserRepository(pool).CreateUser(context.Background(), domain.NewUser{
		Email:        email,
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("create user %q: %v", email, err)
	}
	return user.ID
}

func TestDeckRepositoryCreateDeck(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	userID := createTestUser(t, pool, "owner@mail.ru")

	in := domain.DeckCreate{
		Name:        "Spanish basics",
		Language:    domain.LanguageES,
		Description: "colours and numbers",
	}

	got, err := repo.CreateDeck(ctx, in, userID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	if got.ID <= 0 {
		t.Errorf("ID = %d, want > 0", got.ID)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if loc := got.CreatedAt.Location(); loc != time.UTC {
		t.Errorf("CreatedAt location = %v, want UTC", loc)
	}
	if got.Name != in.Name || got.Language != in.Language || got.Description != in.Description {
		t.Errorf("deck = %+v, want fields from %+v", got, in)
	}

	// The row is really in the table, not just echoed back.
	var (
		name    string
		lang    string
		ownerID int64
	)
	err = pool.QueryRow(ctx, `SELECT name, language, user_id FROM decks WHERE id = $1`, got.ID).
		Scan(&name, &lang, &ownerID)
	if err != nil {
		t.Fatalf("select inserted row: %v", err)
	}
	if name != in.Name || lang != string(in.Language) {
		t.Errorf("stored row = (%q, %q), want (%q, %q)", name, lang, in.Name, in.Language)
	}
	if ownerID != userID {
		t.Errorf("stored user_id = %d, want %d", ownerID, userID)
	}
}

func TestDeckRepositoryCreateDeckUnknownUser(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)

	_, err := repo.CreateDeck(context.Background(), domain.DeckCreate{
		Name:     "Orphan",
		Language: domain.LanguageEN,
	}, 999)
	if err == nil {
		t.Fatal("CreateDeck() for a missing user: want foreign key error, got nil")
	}
}

func TestDeckRepositoryGetDeckByIDOwnership(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")
	strangerID := createTestUser(t, pool, "stranger@mail.ru")

	created, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	got, err := repo.GetDeckByID(ctx, created.ID, ownerID)
	if err != nil {
		t.Fatalf("GetDeckByID() by owner error = %v", err)
	}
	if got.ID != created.ID || got.Name != created.Name {
		t.Errorf("deck = %+v, want %+v", got, created)
	}

	// Someone else's deck must look exactly like a missing one, so ids can't be probed.
	if _, err := repo.GetDeckByID(ctx, created.ID, strangerID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetDeckByID() by stranger error = %v, want %v", err, domain.ErrNotFound)
	}
	if _, err := repo.GetDeckByID(ctx, created.ID+1000, ownerID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetDeckByID() missing id error = %v, want %v", err, domain.ErrNotFound)
	}
}

func TestDeckRepositoryGetDeckListOwnership(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	aliceID := createTestUser(t, pool, "alice@mail.ru")
	bobID := createTestUser(t, pool, "bob@mail.ru")
	emptyID := createTestUser(t, pool, "empty@mail.ru")

	for _, name := range []string{"Alice 1", "Alice 2"} {
		if _, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: name, Language: domain.LanguageEN}, aliceID); err != nil {
			t.Fatalf("CreateDeck(%q) error = %v", name, err)
		}
	}
	if _, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: "Bob 1", Language: domain.LanguageES}, bobID); err != nil {
		t.Fatalf("CreateDeck(Bob 1) error = %v", err)
	}

	tests := []struct {
		name      string
		userID    int64
		wantNames map[string]bool
	}{
		{"alice sees only her decks", aliceID, map[string]bool{"Alice 1": true, "Alice 2": true}},
		{"bob sees only his deck", bobID, map[string]bool{"Bob 1": true}},
		{"user without decks gets empty list", emptyID, map[string]bool{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetDeckList(ctx, tt.userID)
			if err != nil {
				t.Fatalf("GetDeckList() error = %v", err)
			}
			if got == nil {
				t.Fatal("GetDeckList() returned nil slice, want non-nil for [] contract")
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("GetDeckList() len = %d, want %d: %+v", len(got), len(tt.wantNames), got)
			}
			for _, d := range got {
				if !tt.wantNames[d.Name] {
					t.Errorf("unexpected deck %q in list of user %d", d.Name, tt.userID)
				}
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func TestDeckRepositoryUpdateDeckByID(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	tests := []struct {
		name            string
		in              domain.DeckUpdate
		wantName        string
		wantDescription string
	}{
		{
			name:            "only name keeps description",
			in:              domain.DeckUpdate{Name: strPtr("Renamed")},
			wantName:        "Renamed",
			wantDescription: "colours and numbers",
		},
		{
			name:            "only description keeps name",
			in:              domain.DeckUpdate{Description: strPtr("new description")},
			wantName:        "Spanish basics",
			wantDescription: "new description",
		},
		{
			name:            "empty description clears it",
			in:              domain.DeckUpdate{Description: strPtr("")},
			wantName:        "Spanish basics",
			wantDescription: "",
		},
		{
			name:            "both fields",
			in:              domain.DeckUpdate{Name: strPtr("Renamed"), Description: strPtr("new description")},
			wantName:        "Renamed",
			wantDescription: "new description",
		},
		{
			// VARCHAR(100) counts characters, not bytes: 100 cyrillic runes are 200 bytes.
			name:            "name of exactly 100 cyrillic runes fits the column",
			in:              domain.DeckUpdate{Name: strPtr(strings.Repeat("я", 100))},
			wantName:        strings.Repeat("я", 100),
			wantDescription: "colours and numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := repo.CreateDeck(ctx, domain.DeckCreate{
				Name:        "Spanish basics",
				Language:    domain.LanguageES,
				Description: "colours and numbers",
			}, ownerID)
			if err != nil {
				t.Fatalf("CreateDeck() error = %v", err)
			}

			got, err := repo.UpdateDeckByID(ctx, created.ID, ownerID, tt.in)
			if err != nil {
				t.Fatalf("UpdateDeckByID() error = %v", err)
			}
			if got.Name != tt.wantName || got.Description != tt.wantDescription {
				t.Errorf("returned (%q, %q), want (%q, %q)", got.Name, got.Description, tt.wantName, tt.wantDescription)
			}
			// Fields that PATCH can't touch come back as they were.
			if got.ID != created.ID || got.Language != created.Language || !got.CreatedAt.Equal(created.CreatedAt) {
				t.Errorf("returned %+v, want id/language/created_at from %+v", got, created)
			}

			stored, err := repo.GetDeckByID(ctx, created.ID, ownerID)
			if err != nil {
				t.Fatalf("GetDeckByID() error = %v", err)
			}
			if stored.Name != tt.wantName || stored.Description != tt.wantDescription {
				t.Errorf("stored (%q, %q), want (%q, %q)", stored.Name, stored.Description, tt.wantName, tt.wantDescription)
			}
		})
	}
}

func TestDeckRepositoryUpdateDeckByIDKeepsCardCounters(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	created, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	// One card is due, one is scheduled for tomorrow.
	_, err = pool.Exec(ctx, `INSERT INTO cards (deck_id, front, back, due_at)
		VALUES ($1, 'uno', 'one', now() - interval '1 hour'), ($1, 'dos', 'two', now() + interval '1 day')`, created.ID)
	if err != nil {
		t.Fatalf("insert cards: %v", err)
	}

	got, err := repo.UpdateDeckByID(ctx, created.ID, ownerID, domain.DeckUpdate{Name: strPtr("Renamed")})
	if err != nil {
		t.Fatalf("UpdateDeckByID() error = %v", err)
	}
	if got.TotalCards != 2 || got.DueCards != 1 {
		t.Errorf("counters = (total %d, due %d), want (2, 1)", got.TotalCards, got.DueCards)
	}
}

func TestDeckRepositoryUpdateDeckByIDOwnership(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")
	strangerID := createTestUser(t, pool, "stranger@mail.ru")

	created, err := repo.CreateDeck(ctx, domain.DeckCreate{
		Name:        "Spanish",
		Language:    domain.LanguageES,
		Description: "mine",
	}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	tests := []struct {
		name   string
		deckID int64
		userID int64
	}{
		{"someone else's deck", created.ID, strangerID},
		{"missing deck", created.ID + 1000, ownerID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.UpdateDeckByID(ctx, tt.deckID, tt.userID, domain.DeckUpdate{
				Name:        strPtr("Hijacked"),
				Description: strPtr(""),
			})
			if !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("UpdateDeckByID() error = %v, want %v", err, domain.ErrNotFound)
			}
		})
	}

	// The owner's deck must be untouched after the failed attempts.
	stored, err := repo.GetDeckByID(ctx, created.ID, ownerID)
	if err != nil {
		t.Fatalf("GetDeckByID() error = %v", err)
	}
	if stored.Name != "Spanish" || stored.Description != "mine" {
		t.Errorf("stored (%q, %q), want (%q, %q)", stored.Name, stored.Description, "Spanish", "mine")
	}
}

func TestDeckRepositoryDeleteDeckByID(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")

	deleted, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	kept, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: "English", Language: domain.LanguageEN}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO cards (deck_id, front, back, due_at) VALUES ($1, 'uno', 'one', now())`, deleted.ID)
	if err != nil {
		t.Fatalf("insert card: %v", err)
	}

	if err := repo.DeleteDeckByID(ctx, deleted.ID, ownerID); err != nil {
		t.Fatalf("DeleteDeckByID() error = %v", err)
	}

	if _, err := repo.GetDeckByID(ctx, deleted.ID, ownerID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetDeckByID() after delete error = %v, want %v", err, domain.ErrNotFound)
	}
	// The spec promises the deck's cards go with it (ON DELETE CASCADE).
	var cards int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM cards WHERE deck_id = $1`, deleted.ID).Scan(&cards); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if cards != 0 {
		t.Errorf("cards left after delete = %d, want 0", cards)
	}
	if _, err := repo.GetDeckByID(ctx, kept.ID, ownerID); err != nil {
		t.Errorf("GetDeckByID() for the other deck error = %v, want it kept", err)
	}

	// Deleting twice is a 404, not a silent success.
	if err := repo.DeleteDeckByID(ctx, deleted.ID, ownerID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second DeleteDeckByID() error = %v, want %v", err, domain.ErrNotFound)
	}
}

func TestDeckRepositoryDeleteDeckByIDOwnership(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()
	ownerID := createTestUser(t, pool, "owner@mail.ru")
	strangerID := createTestUser(t, pool, "stranger@mail.ru")

	created, err := repo.CreateDeck(ctx, domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES}, ownerID)
	if err != nil {
		t.Fatalf("CreateDeck() error = %v", err)
	}

	tests := []struct {
		name   string
		deckID int64
		userID int64
	}{
		{"someone else's deck", created.ID, strangerID},
		{"missing deck", created.ID + 1000, ownerID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := repo.DeleteDeckByID(ctx, tt.deckID, tt.userID); !errors.Is(err, domain.ErrNotFound) {
				t.Errorf("DeleteDeckByID() error = %v, want %v", err, domain.ErrNotFound)
			}
		})
	}

	// The owner's deck survives the stranger's attempt.
	if _, err := repo.GetDeckByID(ctx, created.ID, ownerID); err != nil {
		t.Errorf("GetDeckByID() by owner error = %v, want the deck kept", err)
	}
}
