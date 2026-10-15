//go:build integration

package postgres

import (
	"context"
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
	_, err := pool.Exec(context.Background(), `TRUNCATE reviews, cards, decks RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func TestDeckRepositoryCreateDeck(t *testing.T) {
	pool := newTestPool(t)
	repo := NewDeckRepository(pool)
	ctx := context.Background()

	in := domain.DeckCreate{
		Name:        "Spanish basics",
		Language:    domain.LanguageES,
		Description: "colours and numbers",
	}

	got, err := repo.CreateDeck(ctx, in)
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
		name string
		lang string
	)
	err = pool.QueryRow(ctx, `SELECT name, language FROM decks WHERE id = $1`, got.ID).
		Scan(&name, &lang)
	if err != nil {
		t.Fatalf("select inserted row: %v", err)
	}
	if name != in.Name || lang != string(in.Language) {
		t.Errorf("stored row = (%q, %q), want (%q, %q)", name, lang, in.Name, in.Language)
	}
}
