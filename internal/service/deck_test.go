package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/ilushka-off/flashcards/internal/domain"
)

// fakeDeckRepository implements DeckRepository without a database. It records
// the input it received and how many times it was called, so tests can assert
// that validation short-circuits before the repository is touched.
type fakeDeckRepository struct {
	gotInput  domain.DeckCreate
	gotUpdate domain.DeckUpdate
	gotDeckID int64
	gotUserID int64
	calls     int
	deck      domain.Deck
	deckList  []domain.Deck
	err       error
}

func (f *fakeDeckRepository) CreateDeck(_ context.Context, in domain.DeckCreate, userID int64) (domain.Deck, error) {
	f.calls++
	f.gotInput = in
	f.gotUserID = userID
	if f.err != nil {
		return domain.Deck{}, f.err
	}
	return f.deck, nil
}

func (f *fakeDeckRepository) GetDeckByID(_ context.Context, deckID, userID int64) (domain.Deck, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	if f.err != nil {
		return domain.Deck{}, f.err
	}
	return f.deck, nil
}

func (f *fakeDeckRepository) GetDeckList(_ context.Context, userID int64) ([]domain.Deck, error) {
	f.calls++
	f.gotUserID = userID
	if f.err != nil {
		return nil, f.err
	}
	return f.deckList, nil
}

func (f *fakeDeckRepository) UpdateDeckByID(_ context.Context, deckID, userID int64, in domain.DeckUpdate) (domain.Deck, error) {
	f.calls++
	f.gotDeckID = deckID
	f.gotUserID = userID
	f.gotUpdate = in
	if f.err != nil {
		return domain.Deck{}, f.err
	}
	return f.deck, nil
}

// testUserID owns the decks in these tests; the service must pass it to the repository.
const testUserID int64 = 42

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

			_, err := svc.Create(context.Background(), tt.in, testUserID)

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

			got, err := svc.Create(context.Background(), tt.in, testUserID)
			if err != nil {
				t.Fatalf("Create() unexpected error = %v", err)
			}
			if repo.calls != 1 {
				t.Fatalf("repository called %d times, want 1", repo.calls)
			}
			if repo.gotUserID != testUserID {
				t.Errorf("repository user id = %d, want %d", repo.gotUserID, testUserID)
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
	}, testUserID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want errors.Is %v", err, wantErr)
	}
}

func TestDeckServiceGetByIDSuccess(t *testing.T) {
	want := domain.Deck{ID: 7, Name: "Spanish", Language: domain.LanguageES}
	repo := &fakeDeckRepository{deck: want}
	svc := NewDeckService(repo)

	got, err := svc.GetByID(context.Background(), 7, testUserID)
	if err != nil {
		t.Fatalf("GetByID() unexpected error = %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repository called %d times, want 1", repo.calls)
	}
	if repo.gotDeckID != 7 || repo.gotUserID != testUserID {
		t.Errorf("repository got (deck %d, user %d), want (7, %d)", repo.gotDeckID, repo.gotUserID, testUserID)
	}
	if got != want {
		t.Errorf("GetByID() = %+v, want %+v", got, want)
	}
}

func TestDeckServiceGetByIDNotFound(t *testing.T) {
	repo := &fakeDeckRepository{err: domain.ErrNotFound}
	svc := NewDeckService(repo)

	_, err := svc.GetByID(context.Background(), 404, testUserID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want errors.Is %v", err, domain.ErrNotFound)
	}
}

func TestDeckServiceGetList(t *testing.T) {
	tests := []struct {
		name     string
		repoList []domain.Deck
		wantLen  int
	}{
		{
			name: "passes decks through",
			repoList: []domain.Deck{
				{ID: 1, Name: "Spanish", Language: domain.LanguageES},
				{ID: 2, Name: "English", Language: domain.LanguageEN},
			},
			wantLen: 2,
		},
		{
			name:     "empty slice stays empty",
			repoList: []domain.Deck{},
			wantLen:  0,
		},
		{
			name:     "nil from repository becomes non-nil empty slice",
			repoList: nil,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeDeckRepository{deckList: tt.repoList}
			svc := NewDeckService(repo)

			got, err := svc.GetList(context.Background(), testUserID)
			if err != nil {
				t.Fatalf("GetList() unexpected error = %v", err)
			}
			if repo.gotUserID != testUserID {
				t.Errorf("repository user id = %d, want %d", repo.gotUserID, testUserID)
			}
			if got == nil {
				t.Fatalf("GetList() returned nil slice, want non-nil for [] contract")
			}
			if len(got) != tt.wantLen {
				t.Errorf("GetList() len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestDeckServiceGetListRepositoryError(t *testing.T) {
	wantErr := errors.New("select failed")
	repo := &fakeDeckRepository{err: wantErr}
	svc := NewDeckService(repo)

	_, err := svc.GetList(context.Background(), testUserID)
	if !errors.Is(err, wantErr) {
		t.Fatalf("GetList() error = %v, want errors.Is %v", err, wantErr)
	}
}

func strPtr(s string) *string { return &s }

// fmtPtr renders an optional field for error messages: nil and "" must be told apart.
func fmtPtr(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return strconv.Quote(*p)
}

func TestDeckServiceUpdateByIDValidation(t *testing.T) {
	tests := []struct {
		name        string
		in          domain.DeckUpdate
		wantField   string
		wantMessage string
	}{
		{
			name:        "empty patch is rejected",
			in:          domain.DeckUpdate{},
			wantField:   "body",
			wantMessage: "нужно передать хотя бы одно поле: name, description",
		},
		{
			name:        "empty name is rejected",
			in:          domain.DeckUpdate{Name: strPtr("")},
			wantField:   "name",
			wantMessage: "не может быть пустым",
		},
		{
			name:        "name of spaces is rejected",
			in:          domain.DeckUpdate{Name: strPtr(" \t\n ")},
			wantField:   "name",
			wantMessage: "не может быть пустым",
		},
		{
			name:        "name of 101 cyrillic runes is too long",
			in:          domain.DeckUpdate{Name: strPtr(strings.Repeat("я", 101))},
			wantField:   "name",
			wantMessage: "не длиннее 100 символов",
		},
		{
			name:        "description of 501 runes is too long",
			in:          domain.DeckUpdate{Description: strPtr(strings.Repeat("д", 501))},
			wantField:   "description",
			wantMessage: "не длиннее 500 символов",
		},
		{
			name:        "invalid name fails even with valid description",
			in:          domain.DeckUpdate{Name: strPtr("  "), Description: strPtr("ok")},
			wantField:   "name",
			wantMessage: "не может быть пустым",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeDeckRepository{}
			svc := NewDeckService(repo)

			_, err := svc.UpdateByID(context.Background(), 7, testUserID, tt.in)

			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("UpdateByID() error = %v, want *domain.ValidationError", err)
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

func TestDeckServiceUpdateByIDSuccess(t *testing.T) {
	tests := []struct {
		name string
		in   domain.DeckUpdate
		// what the repository should receive: trimmed values, nil for untouched fields
		wantName        *string
		wantDescription *string
	}{
		{
			name:     "only name leaves description untouched",
			in:       domain.DeckUpdate{Name: strPtr("Spanish")},
			wantName: strPtr("Spanish"),
		},
		{
			name:            "only description leaves name untouched",
			in:              domain.DeckUpdate{Description: strPtr("colours")},
			wantDescription: strPtr("colours"),
		},
		{
			name:            "empty description clears it instead of being ignored",
			in:              domain.DeckUpdate{Description: strPtr("")},
			wantDescription: strPtr(""),
		},
		{
			name:            "description of spaces is trimmed to empty",
			in:              domain.DeckUpdate{Description: strPtr("   ")},
			wantDescription: strPtr(""),
		},
		{
			name:            "trims both fields",
			in:              domain.DeckUpdate{Name: strPtr("  Spanish basics  "), Description: strPtr("  colours  ")},
			wantName:        strPtr("Spanish basics"),
			wantDescription: strPtr("colours"),
		},
		{
			name:     "name of exactly 100 cyrillic runes is accepted",
			in:       domain.DeckUpdate{Name: strPtr(strings.Repeat("я", 100))},
			wantName: strPtr(strings.Repeat("я", 100)),
		},
		{
			name:     "length is checked after trimming",
			in:       domain.DeckUpdate{Name: strPtr("  " + strings.Repeat("я", 100) + "  ")},
			wantName: strPtr(strings.Repeat("я", 100)),
		},
		{
			name:            "description of exactly 500 runes is accepted",
			in:              domain.DeckUpdate{Description: strPtr(strings.Repeat("д", 500))},
			wantDescription: strPtr(strings.Repeat("д", 500)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := domain.Deck{ID: 7, Name: "Spanish", Language: domain.LanguageES}
			repo := &fakeDeckRepository{deck: want}
			svc := NewDeckService(repo)

			got, err := svc.UpdateByID(context.Background(), 7, testUserID, tt.in)
			if err != nil {
				t.Fatalf("UpdateByID() unexpected error = %v", err)
			}
			if repo.calls != 1 {
				t.Fatalf("repository called %d times, want 1", repo.calls)
			}
			if repo.gotDeckID != 7 || repo.gotUserID != testUserID {
				t.Errorf("repository got (deck %d, user %d), want (7, %d)", repo.gotDeckID, repo.gotUserID, testUserID)
			}
			if fmtPtr(repo.gotUpdate.Name) != fmtPtr(tt.wantName) {
				t.Errorf("repository name = %s, want %s", fmtPtr(repo.gotUpdate.Name), fmtPtr(tt.wantName))
			}
			if fmtPtr(repo.gotUpdate.Description) != fmtPtr(tt.wantDescription) {
				t.Errorf("repository description = %s, want %s", fmtPtr(repo.gotUpdate.Description), fmtPtr(tt.wantDescription))
			}
			if got != want {
				t.Errorf("UpdateByID() = %+v, want %+v", got, want)
			}
		})
	}
}

// The service must not mutate the caller's strings when trimming.
func TestDeckServiceUpdateByIDDoesNotMutateInput(t *testing.T) {
	name := "  Spanish  "
	svc := NewDeckService(&fakeDeckRepository{})

	if _, err := svc.UpdateByID(context.Background(), 7, testUserID, domain.DeckUpdate{Name: &name}); err != nil {
		t.Fatalf("UpdateByID() unexpected error = %v", err)
	}
	if name != "  Spanish  " {
		t.Errorf("caller's name changed to %q", name)
	}
}

func TestDeckServiceUpdateByIDRepositoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		// someone else's deck and a missing one look the same to the service
		{"not found or foreign deck", domain.ErrNotFound},
		{"internal error", errors.New("update failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDeckService(&fakeDeckRepository{err: tt.repoErr})

			_, err := svc.UpdateByID(context.Background(), 7, testUserID, domain.DeckUpdate{Name: strPtr("Spanish")})
			if !errors.Is(err, tt.repoErr) {
				t.Fatalf("UpdateByID() error = %v, want errors.Is %v", err, tt.repoErr)
			}
		})
	}
}
