package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, in domain.UserCreate) (domain.User, error) {
	var user domain.User

	query := `INSERT INTO users(email, password_hash) VALUES ($1, $2) RETURNING id, email, created_at`
	err := r.pool.QueryRow(ctx, query, in.Email, in.Password).
		Scan(&user.ID, &user.Email, &user.CreatedAt)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return domain.User{}, domain.ErrEmailExists
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}
