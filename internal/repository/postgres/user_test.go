package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestUserRepositoryCreateUser(t *testing.T) {
	ctx := context.Background()
	repository := NewUserRepository(testPool(t))

	user, err := repository.CreateUser(ctx, domain.NewUser{Email: "test@mail.ru", PasswordHash: "test"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("returned user id is zero")
	}
	if user.Email != "test@mail.ru" {
		t.Fatalf("email = %q, want %q", user.Email, "test@mail.ru")
	}
	if user.CreatedAt.IsZero() {
		t.Fatalf("created_at is zero")
	}

	t.Run("duplicate email", func(t *testing.T) {
		_, err := repository.CreateUser(ctx, domain.NewUser{
			Email:        "test@mail.ru",
			PasswordHash: "test",
		})
		if !errors.Is(err, domain.ErrEmailExists) {
			t.Fatalf("error : %v, want %v", err, domain.ErrEmailExists)
		}
	})

	t.Run("email longer than the column", func(t *testing.T) {
		_, err := repository.CreateUser(ctx, domain.NewUser{
			Email:        strings.Repeat("a", 256),
			PasswordHash: "test",
		})
		if err == nil {
			t.Fatal("expected error for an oversized email")
		}
	})

}
