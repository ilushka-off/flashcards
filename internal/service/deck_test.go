package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// fakeDeckRepository implements DeckRepository without a database. It records
// the input it received and how many times it was called, so tests can assert
// that validation short-circuits before the repository is touched.
type fakeDeckRepository struct {
	gotInput domain.DeckCreate
	calls    int
	deck     domain.Deck
	err      error
}

func (f *fakeDeckRepository) CreateDeck(_ context.Context, in domain.DeckCreate) (domain.Deck, error) {
	f.calls++
	f.gotInput = in
	if f.err != nil {
		return domain.Deck{}, f.err
	}
	return f.deck, nil
}

func (f *fakeDeckRepository) GetDeckByID(_ context.Context, _ int64) (domain.Deck, error) {
	return domain.Deck{}, nil
}

func (f *fakeDeckRepository) GetDeckList(_ context.Context) ([]domain.Deck, error) {
	return nil, nil
}

func TestDeckServiceCreateValidation(t *testing.T) {
	tests := []struct {
		name        string
		in          domain.DeckCreate
		wantField   string
		wantMessage string
	}{
		{
			name:        "blank name is rejected",
			in:          domain.DeckCreate{Name: "   ", Language: domain.LanguageEN},
			wantField:   "name",
			wantMessage: "обязательное поле",
		},
		{
			name:        "name of 101 cyrillic runes is too long",
			in:          domain.DeckCreate{Name: strings.Repeat("я", 101), Language: domain.LanguageEN},
			wantField:   "name",
			wantMessage: "не длиннее 100 символов",
		},
		{
			name:        "unknown language is rejected",
			in:          domain.DeckCreate{Name: "Spanish", Language: "fr"},
			wantField:   "language",
			wantMessage: "допустимые значения en, es",
		},
		{
			name:        "description over 500 runes is too long",
			in:          domain.DeckCreate{Name: "Spanish", Language: domain.LanguageES, Description: strings.Repeat("д", 501)},
			wantField:   "description",
			wantMessage: "не длиннее 500 символов",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeDeckRepository{}
			svc := NewDeckService(repo)

			_, err := svc.Create(context.Background(), tt.in)

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

func TestDeckServiceCreateSuccess(t *testing.T) {
	tests := []struct {
		name      string
		in        domain.DeckCreate
		wantInput domain.DeckCreate // what the repository should receive after trimming
	}{
		{
			name: "trims name and description before storing",
			in: domain.DeckCreate{
				Name:        "  Spanish basics  ",
				Language:    domain.LanguageES,
				Description: "  colours and numbers  ",
			},
			wantInput: domain.DeckCreate{
				Name:        "Spanish basics",
				Language:    domain.LanguageES,
				Description: "colours and numbers",
			},
		},
		{
			name: "name of exactly 100 cyrillic runes is accepted",
			in: domain.DeckCreate{
				Name:     strings.Repeat("я", 100),
				Language: domain.LanguageEN,
			},
			wantInput: domain.DeckCreate{
				Name:     strings.Repeat("я", 100),
				Language: domain.LanguageEN,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := domain.Deck{ID: 7, Name: tt.wantInput.Name, Language: tt.wantInput.Language}
			repo := &fakeDeckRepository{deck: want}
			svc := NewDeckService(repo)

			got, err := svc.Create(context.Background(), tt.in)
			if err != nil {
				t.Fatalf("Create() unexpected error = %v", err)
			}
			if repo.calls != 1 {
				t.Fatalf("repository called %d times, want 1", repo.calls)
			}
			if repo.gotInput != tt.wantInput {
				t.Errorf("repository input = %+v, want %+v", repo.gotInput, tt.wantInput)
			}
			if got != want {
				t.Errorf("Create() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDeckServiceCreateRepositoryError(t *testing.T) {
	wantErr := errors.New("insert failed")
	repo := &fakeDeckRepository{err: wantErr}
	svc := NewDeckService(repo)

	_, err := svc.Create(context.Background(), domain.DeckCreate{
		Name:     "Spanish",
		Language: domain.LanguageES,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want errors.Is %v", err, wantErr)
	}
}
