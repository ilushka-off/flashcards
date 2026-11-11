package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestCardServiceDeleteByID(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		canceled bool
	}{
		{name: "success"},
		{name: "missing or foreign card", err: fmt.Errorf("delete card: %w", domain.ErrNotFound)},
		{name: "database error", err: errors.New("delete failed")},
		{name: "canceled request", err: context.Canceled, canceled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			repo := &fakeCardRepository{err: tt.err}

			err := NewCardService(repo).DeleteByID(ctx, 42, testUserID)

			if err != tt.err {
				t.Errorf("DeleteByID() error = %v, want original error %v", err, tt.err)
			}
			if repo.calls != 1 || repo.gotCardID != 42 || repo.gotUserID != testUserID {
				t.Errorf("repository got (calls %d, card %d, user %d), want (1, 42, %d)", repo.calls, repo.gotCardID, repo.gotUserID, testUserID)
			}
			if repo.gotCtx != ctx {
				t.Error("repository did not receive the request context")
			}
		})
	}
}
