package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ilushka-off/flashcards/internal/auth"
	"github.com/ilushka-off/flashcards/internal/domain"
)

// fakeUserRepository implements UserRepository without a database. It records the
// input it received and the call count, so tests can assert that validation
// short-circuits before the repository is touched.
type fakeUserRepository struct {
	gotInput domain.NewUser
	gotEmail string
	calls    int
	user     domain.User
	err      error
	getUser  domain.User
	getErr   error
}

func (f *fakeUserRepository) CreateUser(_ context.Context, in domain.NewUser) (domain.User, error) {
	f.calls++
	f.gotInput = in
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

func (f *fakeUserRepository) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	f.calls++
	f.gotEmail = email
	if f.getErr != nil {
		return domain.User{}, f.getErr
	}
	return f.getUser, nil
}

// fakeTokenIssuer implements TokenIssuer so tests can force the "token not issued"
// branch without a real signing key.
type fakeTokenIssuer struct {
	gotUserID int64
	calls     int
	token     string
	err       error
}

func (f *fakeTokenIssuer) Issue(userID int64) (string, error) {
	f.calls++
	f.gotUserID = userID
	return f.token, f.err
}

// fakeRefreshStore implements RefreshStore in memory. It records the row passed
// to CreateRefreshToken and which ids/users were revoked, so tests can assert the
// stored hash and the rotation/reuse behaviour without a database.
type fakeRefreshStore struct {
	created     domain.NewRefreshToken
	createCalls int
	createErr   error
	getToken    domain.RefreshToken
	getErr      error
	revokedID   int64
	revokedUser int64
}

func (f *fakeRefreshStore) CreateRefreshToken(_ context.Context, in domain.NewRefreshToken) error {
	f.createCalls++
	f.created = in
	return f.createErr
}

func (f *fakeRefreshStore) GetRefreshTokenByHash(_ context.Context, _ string) (domain.RefreshToken, error) {
	if f.getErr != nil {
		return domain.RefreshToken{}, f.getErr
	}
	return f.getToken, nil
}

func (f *fakeRefreshStore) RevokeRefreshToken(_ context.Context, id int64) error {
	f.revokedID = id
	return nil
}

func (f *fakeRefreshStore) RevokeAllForUser(_ context.Context, userID int64) error {
	f.revokedUser = userID
	return nil
}

const (
	validPassword  = "password123" // 11 bytes, within 8..72
	testRefreshTTL = 30 * 24 * time.Hour
)

func TestUserServiceRegisterValidation(t *testing.T) {
	tests := []struct {
		name string
		in   domain.UserCreate
	}{
		{"blank email", domain.UserCreate{Email: "", Password: validPassword}},
		{"malformed email", domain.UserCreate{Email: "not-an-email", Password: validPassword}},
		{"display-name form", domain.UserCreate{Email: "Ilya <ilya@mail.ru>", Password: validPassword}},
		{"email over 255 runes", domain.UserCreate{Email: strings.Repeat("a", 250) + "@example.com", Password: validPassword}},
		{"password under 8 bytes", domain.UserCreate{Email: "ilya@mail.ru", Password: "short7!"}},
		{"password over 72 bytes", domain.UserCreate{Email: "ilya@mail.ru", Password: strings.Repeat("x", 73)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			issuer := &fakeTokenIssuer{}
			svc := NewUserService(repo, issuer, &fakeRefreshStore{}, testRefreshTTL)

			_, _, err := svc.Register(context.Background(), tt.in)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Register() error = %v, want ErrInvalidInput", err)
			}
			if repo.calls != 0 {
				t.Errorf("repository called %d times, want 0 on validation error", repo.calls)
			}
			if issuer.calls != 0 {
				t.Errorf("issuer called %d times, want 0 on validation error", issuer.calls)
			}
		})
	}
}

func TestUserServiceRegisterSuccess(t *testing.T) {
	stored := domain.User{ID: 42, Email: "ilya@mail.ru"}
	repo := &fakeUserRepository{user: stored}
	issuer := &fakeTokenIssuer{token: "signed-token"}
	svc := NewUserService(repo, issuer, &fakeRefreshStore{}, testRefreshTTL)

	// Mixed case and surrounding spaces must be normalized before storage.
	user, token, err := svc.Register(context.Background(), domain.UserCreate{
		Email:    "  Ilya@Mail.ru ",
		Password: validPassword,
	})
	if err != nil {
		t.Fatalf("Register() unexpected error = %v", err)
	}
	if repo.calls != 1 {
		t.Fatalf("repository called %d times, want 1", repo.calls)
	}
	if repo.gotInput.Email != "ilya@mail.ru" {
		t.Errorf("stored email = %q, want normalized %q", repo.gotInput.Email, "ilya@mail.ru")
	}
	// The repository must receive a bcrypt hash, never the raw password.
	if repo.gotInput.PasswordHash == validPassword || repo.gotInput.PasswordHash == "" {
		t.Fatalf("stored password hash = %q, want a bcrypt hash, not the raw password", repo.gotInput.PasswordHash)
	}
	if err := auth.CheckPassword(repo.gotInput.PasswordHash, validPassword); err != nil {
		t.Errorf("stored hash does not verify against the password: %v", err)
	}
	if issuer.gotUserID != stored.ID {
		t.Errorf("token issued for id %d, want %d", issuer.gotUserID, stored.ID)
	}
	if token != "signed-token" {
		t.Errorf("token = %q, want %q", token, "signed-token")
	}
	if user != stored {
		t.Errorf("user = %+v, want %+v", user, stored)
	}
}

func TestUserServiceRegisterEmailTaken(t *testing.T) {
	repo := &fakeUserRepository{err: domain.ErrEmailExists}
	issuer := &fakeTokenIssuer{}
	svc := NewUserService(repo, issuer, &fakeRefreshStore{}, testRefreshTTL)

	_, _, err := svc.Register(context.Background(), domain.UserCreate{Email: "ilya@mail.ru", Password: validPassword})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("Register() error = %v, want ErrEmailTaken", err)
	}
	if issuer.calls != 0 {
		t.Errorf("issuer called %d times, want 0 when the email is taken", issuer.calls)
	}
}

func TestUserServiceRegisterRepositoryError(t *testing.T) {
	wantErr := errors.New("insert failed")
	repo := &fakeUserRepository{err: wantErr}
	svc := NewUserService(repo, &fakeTokenIssuer{}, &fakeRefreshStore{}, testRefreshTTL)

	_, _, err := svc.Register(context.Background(), domain.UserCreate{Email: "ilya@mail.ru", Password: validPassword})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Register() error = %v, want errors.Is %v", err, wantErr)
	}
	if errors.Is(err, ErrEmailTaken) {
		t.Error("a generic repository error must not map to ErrEmailTaken")
	}
}

func TestUserServiceLoginSuccess(t *testing.T) {
	hash, err := auth.HashPassword(validPassword)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo := &fakeUserRepository{getUser: domain.User{ID: 42, Email: "ilya@mail.ru", PasswordHash: hash}}
	issuer := &fakeTokenIssuer{token: "signed-token"}
	refresh := &fakeRefreshStore{}
	svc := NewUserService(repo, issuer, refresh, testRefreshTTL)

	pair, err := svc.Login(context.Background(), "  Ilya@Mail.ru ", validPassword)
	if err != nil {
		t.Fatalf("Login() unexpected error = %v", err)
	}
	if repo.gotEmail != "ilya@mail.ru" {
		t.Errorf("looked up email = %q, want normalized %q", repo.gotEmail, "ilya@mail.ru")
	}
	if issuer.gotUserID != 42 {
		t.Errorf("token issued for id %d, want 42", issuer.gotUserID)
	}
	if pair.AccessToken != "signed-token" {
		t.Errorf("access token = %q, want %q", pair.AccessToken, "signed-token")
	}
	if pair.RefreshToken == "" {
		t.Fatal("refresh token is empty, want a generated opaque token")
	}
	// The store must receive the sha256 hash of the raw refresh token, never the raw.
	if refresh.createCalls != 1 {
		t.Fatalf("CreateRefreshToken called %d times, want 1", refresh.createCalls)
	}
	wantHash, _ := auth.HashRefreshToken(pair.RefreshToken)
	if refresh.created.TokenHash != wantHash {
		t.Errorf("stored hash = %q, want sha256(raw) = %q", refresh.created.TokenHash, wantHash)
	}
	if refresh.created.TokenHash == pair.RefreshToken {
		t.Error("stored the raw refresh token instead of its sha256 hash")
	}
	if refresh.created.UserID != 42 {
		t.Errorf("stored refresh for user %d, want 42", refresh.created.UserID)
	}
}

func TestUserServiceLoginWrongPassword(t *testing.T) {
	hash, err := auth.HashPassword("the-real-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repo := &fakeUserRepository{getUser: domain.User{ID: 42, PasswordHash: hash}}
	issuer := &fakeTokenIssuer{}
	svc := NewUserService(repo, issuer, &fakeRefreshStore{}, testRefreshTTL)

	_, err = svc.Login(context.Background(), "ilya@mail.ru", validPassword)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if issuer.calls != 0 {
		t.Errorf("issuer called %d times, want 0 on wrong password", issuer.calls)
	}
}

func TestUserServiceLoginUnknownUser(t *testing.T) {
	repo := &fakeUserRepository{getErr: domain.ErrNotFound}
	issuer := &fakeTokenIssuer{}
	svc := NewUserService(repo, issuer, &fakeRefreshStore{}, testRefreshTTL)

	_, err := svc.Login(context.Background(), "ghost@mail.ru", validPassword)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials (same as wrong password)", err)
	}
	if issuer.calls != 0 {
		t.Errorf("issuer called %d times, want 0 for unknown user", issuer.calls)
	}
}

func TestUserServiceLoginValidation(t *testing.T) {
	tests := []struct {
		name, email, password string
	}{
		{"malformed email", "not-an-email", validPassword},
		{"password too short", "ilya@mail.ru", "short7!"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			svc := NewUserService(repo, &fakeTokenIssuer{}, &fakeRefreshStore{}, testRefreshTTL)

			_, err := svc.Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Login() error = %v, want ErrInvalidInput", err)
			}
			if repo.calls != 0 {
				t.Errorf("repository called %d times, want 0 on validation error", repo.calls)
			}
		})
	}
}

func TestUserServiceRegisterTokenNotIssued(t *testing.T) {
	wantErr := errors.New("sign failed")
	repo := &fakeUserRepository{user: domain.User{ID: 7, Email: "ilya@mail.ru"}}
	issuer := &fakeTokenIssuer{err: wantErr}
	svc := NewUserService(repo, issuer, &fakeRefreshStore{}, testRefreshTTL)

	user, token, err := svc.Register(context.Background(), domain.UserCreate{Email: "ilya@mail.ru", Password: validPassword})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Register() error = %v, want errors.Is %v", err, wantErr)
	}
	// On a token failure nothing partial should leak out.
	if token != "" {
		t.Errorf("token = %q, want empty on issue failure", token)
	}
	if user != (domain.User{}) {
		t.Errorf("user = %+v, want zero value on issue failure", user)
	}
}
