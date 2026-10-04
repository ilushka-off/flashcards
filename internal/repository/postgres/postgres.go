package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pingTimeout ограничивает проверку соединения при старте:
// если база не ответила за это время, сервис падает с ошибкой, а не висит.
const (
	pingTimeout = 5 * time.Second
)

func OpenPool(ctx context.Context, databaseURI string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURI)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
