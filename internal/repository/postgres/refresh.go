package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshRepository(pool *pgxpool.Pool) *RefreshRepository {
	return &RefreshRepository{pool: pool}
}

func (r *RefreshRepository) CreateRefreshToken(ctx context.Context, in domain.NewRefreshToken) error {
	query := `INSERT INTO referesh_tokens(user_id, refresh_token, created_at) VALUES ($1, $2, $3)`

	_, err := r.pool.Exec(ctx, query, in.UserID, in.TokenHash, in.ExpiresAt)
	if err != nil {
		return fmt.Errorf("could not create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	var refreshToken domain.RefreshToken

	query := `SELECT id, user_id, token_hash, expires_at, refresh_token FROM refresh_tokens WHERE refresh_token = $1`

	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(&refreshToken.ID, &refreshToken.UserID, &refreshToken.TokenHash, &refreshToken.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RefreshToken{}, fmt.Errorf("refresh token not found: %w", err)
	}
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("could not get refresh token: %w", err)
	}
	return refreshToken, nil
}

func (r *RefreshRepository) RevokeRefreshToken(ctx context.Context, refreshTokenID int64) error {
	query := `UPDATE refresh_tokens SER revoked_at = now() WHERE id = $1 AND revoked_At IS NULL`

	_, err := r.pool.Exec(ctx, query, refreshTokenID)
	if err != nil {
		return fmt.Errorf("could not revoke refresh token: %w", err)
	}
	return nil
}
func (r *RefreshRepository) RevokeAllForUser(ctx context.Context, userID int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_At IS NULL`
	_, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("could not revoke all refresh tokens: %w", err)
	}
	return nil
}
