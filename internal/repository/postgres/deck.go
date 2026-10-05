package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeckRepository struct {
	pool *pgxpool.Pool
}

func NewDeckRepository(pool *pgxpool.Pool) *DeckRepository {
	return &DeckRepository{
		pool: pool,
	}
}

func (r *DeckRepository) CreateDeck(ctx context.Context, in domain.DeckCreate) (domain.Deck, error) {

	var deckID int64
	var createdAt time.Time

	query := `INSERT INTO decks(name, language, description) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, in.Name, in.Language, in.Description).Scan(&deckID, &createdAt)
	if err != nil {
		return domain.Deck{}, fmt.Errorf("create deck: %w", err)
	}
	return domain.Deck{
		ID:          deckID,
		Name:        in.Name,
		Language:    in.Language,
		Description: in.Description,
		TotalCards:  0,
		DueCards:    0,
		CreatedAt:   createdAt.UTC(),
	}, nil
}
