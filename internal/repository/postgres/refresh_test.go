package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestRefreshRepository(t *testing.T) {
	ctx := context.Background()
	pool := testPool(t)
	users := NewUserRepository(pool)
	refresh := NewRefreshRepository(pool)

	// refresh_tokens.user_id has a FK to users(id), so we need a real user first.
	user, err := users.CreateUser(ctx, domain.NewUser{Email: "ilya@mail.ru", PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	expires := time.Now().Add(time.Hour)
	if err := refresh.CreateRefreshToken(ctx, domain.NewRefreshToken{
		UserID:    user.ID,
		TokenHash: "hash-aaa",
		ExpiresAt: expires,
	}); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	t.Run("get by hash returns the stored row", func(t *testing.T) {
		got, err := refresh.GetRefreshTokenByHash(ctx, "hash-aaa")
		if err != nil {
			t.Fatalf("get by hash: %v", err)
		}
		if got.ID == 0 {
			t.Error("ID is zero")
		}
		if got.UserID != user.ID {
			t.Errorf("UserID = %d, want %d", got.UserID, user.ID)
		}
		if got.RevokedAt != nil {
			t.Errorf("RevokedAt = %v, want nil for a fresh token", got.RevokedAt)
		}
		if got.ExpiresAt.Before(time.Now()) {
			t.Error("ExpiresAt is in the past")
		}
	})

	t.Run("missing hash returns ErrNotFound", func(t *testing.T) {
		_, err := refresh.GetRefreshTokenByHash(ctx, "does-not-exist")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("error = %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("revoke marks revoked_at", func(t *testing.T) {
		got, err := refresh.GetRefreshTokenByHash(ctx, "hash-aaa")
		if err != nil {
			t.Fatalf("get before revoke: %v", err)
		}
		if err := refresh.RevokeRefreshToken(ctx, got.ID); err != nil {
			t.Fatalf("revoke: %v", err)
		}
		got, err = refresh.GetRefreshTokenByHash(ctx, "hash-aaa")
		if err != nil {
			t.Fatalf("get after revoke: %v", err)
		}
		if got.RevokedAt == nil {
			t.Error("RevokedAt is nil after revoke, want a timestamp")
		}
	})

	t.Run("revoke all for user revokes every live token", func(t *testing.T) {
		for _, h := range []string{"hash-bbb", "hash-ccc"} {
			if err := refresh.CreateRefreshToken(ctx, domain.NewRefreshToken{
				UserID:    user.ID,
				TokenHash: h,
				ExpiresAt: expires,
			}); err != nil {
				t.Fatalf("create %s: %v", h, err)
			}
		}

		if err := refresh.RevokeAllForUser(ctx, user.ID); err != nil {
			t.Fatalf("revoke all: %v", err)
		}

		for _, h := range []string{"hash-bbb", "hash-ccc"} {
			got, err := refresh.GetRefreshTokenByHash(ctx, h)
			if err != nil {
				t.Fatalf("get %s: %v", h, err)
			}
			if got.RevokedAt == nil {
				t.Errorf("token %s: RevokedAt is nil after RevokeAllForUser", h)
			}
		}
	})
}
