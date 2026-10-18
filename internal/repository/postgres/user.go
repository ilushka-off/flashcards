package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ()

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) CreateUser(ctx context.Context, in domain.UserCreate) (int64, error) {
	var userID int64

	query := `INSERT INTO users(email, password_hash) VALUES ($1, $2) RETURNING id`
	err := r.pool.QueryRow(ctx, query, in.Email, in.Password).Scan(&userID)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		return 0, domain.ErrEmailExists
	}
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	return userID, nil
}
