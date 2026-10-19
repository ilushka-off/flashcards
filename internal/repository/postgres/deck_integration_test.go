//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
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
