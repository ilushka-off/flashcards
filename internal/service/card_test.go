package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// fakeCardRepository implements CardRepository without a database. It records
// what it received so tests can check what the service passed on.
type fakeCardRepository struct {
	gotInput  domain.CardCreate
	gotDeckID int64
	gotUserID int64
	calls     int
	err       error
}

func (f *fakeCardRepository) CreateCard(_ context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	f.gotInput = in
	if f.err != nil {
		return domain.Card{}, f.err
	}
	return domain.Card{ID: 1, DeckID: deckID, Front: in.Front, Back: in.Back, Example: in.Example}, nil
}

func TestCardServiceCreateValidation(t *testing.T) {
	tests := []struct {
		name        string
		in          domain.CardCreate
		wantField   string
		wantMessage string
	}{
		{
			name:        "blank front is rejected",
			in:          domain.CardCreate{Front: "   ", Back: "uno"},
			wantField:   "front",
			wantMessage: "обязательное поле",
		},
		{
			name:        "front of 201 cyrillic runes is too long",
			in:          domain.CardCreate{Front: strings.Repeat("я", 201), Back: "uno"},
			wantField:   "front",
			wantMessage: "не длиннее 200 символов",
		},
		{
			name:        "blank back is rejected",
			in:          domain.CardCreate{Front: "one", Back: ""},
			wantField:   "back",
			wantMessage: "обязательное поле",
		},
		{
			name:        "back of 201 runes is too long",
			in:          domain.CardCreate{Front: "one", Back: strings.Repeat("b", 201)},
			wantField:   "back",
			wantMessage: "не длиннее 200 символов",
		},
		{
			name:        "example over 500 runes is too long",
			in:          domain.CardCreate{Front: "one", Back: "uno", Example: strings.Repeat("п", 501)},
			wantField:   "example",
			wantMessage: "не длиннее 500 символов",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeCardRepository{}
			svc := NewCardService(repo)

			_, err := svc.Create(context.Background(), 7, testUserID, tt.in)

			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Create() error = %v, want *domain.ValidationError", err)
			}
			if verr.Field != tt.wantField {
				t.Errorf("Field = %q, want %q", verr.Field, tt.wantField)
			}
			if verr.Message != tt.wantMessage {
				t.Errorf("Message = %q, want %q", verr.Message, tt.wantMessage)
			}
			if repo.calls != 0 {
				t.Errorf("repository called %d times, want 0 on validation error", repo.calls)
			}
		})
	}
}

func TestCardServiceCreateSuccess(t *testing.T) {
	repo := &fakeCardRepository{}
	svc := NewCardService(repo)

	in := domain.CardCreate{
		Front:   "  serendipity ",
		Back:    "счастливая случайность",
		Example: " It was pure serendipity. ",
	}
	got, err := svc.Create(context.Background(), 7, testUserID, in)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	wantInput := domain.CardCreate{
		Front:   "serendipity",
		Back:    "счастливая случайность",
		Example: "It was pure serendipity.",
	}
	if repo.gotInput != wantInput {
		t.Errorf("repository got %+v, want %+v", repo.gotInput, wantInput)
	}
	if repo.gotDeckID != 7 || repo.gotUserID != testUserID {
		t.Errorf("repository got (deck %d, user %d), want (7, %d)", repo.gotDeckID, repo.gotUserID, testUserID)
	}
	// The fake assigns ID 1, as the database would; the service must return it.
	if got.ID != 1 || got.DeckID != 7 {
		t.Errorf("card = %+v, want ID 1 in deck 7", got)
	}
}

func TestCardServiceCreateBoundaryLengths(t *testing.T) {
	in := domain.CardCreate{
		Front:   strings.Repeat("я", 200),
		Back:    strings.Repeat("b", 200),
		Example: strings.Repeat("п", 500),
	}
	repo := &fakeCardRepository{}
	svc := NewCardService(repo)

	if _, err := svc.Create(context.Background(), 7, testUserID, in); err != nil {
		t.Fatalf("Create() error = %v, want nil at max lengths", err)
	}
	if repo.calls != 1 {
		t.Errorf("repository called %d times, want 1", repo.calls)
	}
}

func TestCardServiceCreatePassesRepositoryError(t *testing.T) {
	repo := &fakeCardRepository{err: domain.ErrNotFound}
	svc := NewCardService(repo)

	_, err := svc.Create(context.Background(), 7, testUserID, domain.CardCreate{Front: "one", Back: "uno"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Create() error = %v, want ErrNotFound", err)
	}
}

func TestCardServiceCreateExample(t *testing.T) {
	tests := []struct {
		name    string
		example string
		want    string
	}{
		{name: "omitted example stays empty", example: "", want: ""},
		{name: "whitespace-only example becomes empty", example: "   ", want: ""},
		{name: "example is trimmed", example: "  uno, dos  ", want: "uno, dos"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeCardRepository{}
			svc := NewCardService(repo)

			_, err := svc.Create(context.Background(), 7, testUserID, domain.CardCreate{Front: "one", Back: "uno", Example: tt.example})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if repo.gotInput.Example != tt.want {
				t.Errorf("Example = %q, want %q", repo.gotInput.Example, tt.want)
			}
		})
	}
}
