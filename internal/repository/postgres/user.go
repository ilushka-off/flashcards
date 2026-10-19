package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, in domain.NewUser) (domain.User, error) {
	var user domain.User

	query := `INSERT INTO users(email, password_hash) VALUES ($1, $2) RETURNING id, email, created_at`
	err := r.pool.QueryRow(ctx, query, in.Email, in.PasswordHash).
		Scan(&user.ID, &user.Email, &user.CreatedAt)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return domain.User{}, domain.ErrEmailExists
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User

	query := `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`

	err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return user, nil
}
