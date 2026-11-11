package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
)

func TestCardServiceUpdatePartial(t *testing.T) {
	tests := []struct {
		name string
		in   domain.CardUpdate
		want domain.CardUpdate
	}{
		{
			name: "front only",
			in:   domain.CardUpdate{Front: strPtr("  uno  ")},
			want: domain.CardUpdate{Front: strPtr("uno")},
		},
		{
			name: "back only",
			in:   domain.CardUpdate{Back: strPtr("  один  ")},
			want: domain.CardUpdate{Back: strPtr("один")},
		},
		{
			name: "example only",
			in:   domain.CardUpdate{Example: strPtr("  uno, dos  ")},
			want: domain.CardUpdate{Example: strPtr("uno, dos")},
		},
		{
			name: "back and example without front reproduces reported panic",
			in:   domain.CardUpdate{Back: strPtr("  один  "), Example: strPtr("  uno, dos  ")},
			want: domain.CardUpdate{Back: strPtr("один"), Example: strPtr("uno, dos")},
		},
		{
			name: "empty example clears it",
			in:   domain.CardUpdate{Example: strPtr("")},
			want: domain.CardUpdate{Example: strPtr("")},
		},
		{
			name: "blank example clears it after trimming",
			in:   domain.CardUpdate{Example: strPtr(" \t\n ")},
			want: domain.CardUpdate{Example: strPtr("")},
		},
		{
			name: "all fields",
			in:   domain.CardUpdate{Front: strPtr(" uno "), Back: strPtr(" один "), Example: strPtr(" пример ")},
			want: domain.CardUpdate{Front: strPtr("uno"), Back: strPtr("один"), Example: strPtr("пример")},
		},
		{
			name: "single character after trimming",
			in:   domain.CardUpdate{Front: strPtr(" я "), Back: strPtr(" я ")},
			want: domain.CardUpdate{Front: strPtr("я"), Back: strPtr("я")},
		},
		{
			name: "maximum character lengths after trimming",
			in: domain.CardUpdate{
				Front:   strPtr(" " + strings.Repeat("я", 200) + " "),
				Back:    strPtr(" " + strings.Repeat("я", 200) + " "),
				Example: strPtr(" " + strings.Repeat("я", 500) + " "),
			},
			want: domain.CardUpdate{
				Front:   strPtr(strings.Repeat("я", 200)),
				Back:    strPtr(strings.Repeat("я", 200)),
				Example: strPtr(strings.Repeat("я", 500)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantCard := domain.Card{ID: 42, DeckID: 7, Front: "uno", Back: "один", EaseFactor: 2.15, IntervalDays: 14, Repetitions: 4}
			repo := &fakeCardRepository{card: wantCard}
			svc := NewCardService(repo)
			original := [3]string{fmtPtr(tt.in.Front), fmtPtr(tt.in.Back), fmtPtr(tt.in.Example)}
			before := time.Now().UTC()

			got, err := svc.UpdateByID(context.Background(), 42, testUserID, tt.in)
			after := time.Now().UTC()
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}
			if repo.calls != 1 || repo.gotCardID != 42 || repo.gotUserID != testUserID {
				t.Errorf("repository got (calls %d, card %d, user %d), want (1, 42, %d)", repo.calls, repo.gotCardID, repo.gotUserID, testUserID)
			}
			for name, values := range map[string][2]*string{
				"front":   {repo.gotUpdate.Front, tt.want.Front},
				"back":    {repo.gotUpdate.Back, tt.want.Back},
				"example": {repo.gotUpdate.Example, tt.want.Example},
			} {
				if fmtPtr(values[0]) != fmtPtr(values[1]) {
					t.Errorf("repository %s = %s, want %s", name, fmtPtr(values[0]), fmtPtr(values[1]))
				}
			}
			if repo.gotNow.Before(before) || repo.gotNow.After(after) || repo.gotNow.Location() != time.UTC {
				t.Errorf("repository now = %v, want UTC time between %v and %v", repo.gotNow, before, after)
			}
			if got != wantCard {
				t.Errorf("Update() card = %+v, want unchanged repository result %+v", got, wantCard)
			}
			if afterInput := [3]string{fmtPtr(tt.in.Front), fmtPtr(tt.in.Back), fmtPtr(tt.in.Example)}; afterInput != original {
				t.Errorf("input was mutated: got %v, want %v", afterInput, original)
			}
		})
	}
}

func TestCardServiceUpdateValidation(t *testing.T) {
	tests := []struct {
		name        string
		in          domain.CardUpdate
		wantField   string
		wantMessage string
	}{
		{name: "empty patch", wantField: "body", wantMessage: "нужно передать хотя бы одно поле: front, back, example"},
		{name: "empty front", in: domain.CardUpdate{Front: strPtr("")}, wantField: "front", wantMessage: "обязательное поле"},
		{name: "blank front", in: domain.CardUpdate{Front: strPtr(" \t\n ")}, wantField: "front", wantMessage: "обязательное поле"},
		{name: "empty back", in: domain.CardUpdate{Back: strPtr("")}, wantField: "back", wantMessage: "обязательное поле"},
		{name: "blank back", in: domain.CardUpdate{Back: strPtr(" \t\n ")}, wantField: "back", wantMessage: "обязательное поле"},
		{name: "front too long", in: domain.CardUpdate{Front: strPtr(strings.Repeat("я", 201))}, wantField: "front", wantMessage: "не длиннее 200 символов"},
		{name: "back too long", in: domain.CardUpdate{Back: strPtr(strings.Repeat("я", 201))}, wantField: "back", wantMessage: "не длиннее 200 символов"},
		{name: "example too long", in: domain.CardUpdate{Example: strPtr(strings.Repeat("я", 501))}, wantField: "example", wantMessage: "не длиннее 500 символов"},
		{name: "valid front with invalid back", in: domain.CardUpdate{Front: strPtr("uno"), Back: strPtr("")}, wantField: "back", wantMessage: "обязательное поле"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeCardRepository{}
			got, err := NewCardService(repo).UpdateByID(context.Background(), 42, testUserID, tt.in)
			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Update() error = %v, want ValidationError", err)
			}
			if verr.Field != tt.wantField || verr.Message != tt.wantMessage {
				t.Errorf("error = %q: %q, want %q: %q", verr.Field, verr.Message, tt.wantField, tt.wantMessage)
			}
			if repo.calls != 0 {
				t.Errorf("repository called %d times, want 0", repo.calls)
			}
			if got != (domain.Card{}) {
				t.Errorf("card on error = %+v, want zero value", got)
			}
		})
	}
}

func TestCardServiceUpdateRepositoryErrors(t *testing.T) {
	for _, wantErr := range []error{domain.ErrNotFound, errors.New("update failed")} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			repo := &fakeCardRepository{err: wantErr}
			got, err := NewCardService(repo).UpdateByID(context.Background(), 42, testUserID, domain.CardUpdate{Front: strPtr("uno")})
			if !errors.Is(err, wantErr) {
				t.Errorf("Update() error = %v, want %v", err, wantErr)
			}
			if got != (domain.Card{}) {
				t.Errorf("card on error = %+v, want zero value", got)
			}
		})
	}
}
