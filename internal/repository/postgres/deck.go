package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5"
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

func (r *DeckRepository) GetDeckList(ctx context.Context) ([]domain.Deck, error) {
	decks := make([]domain.Deck, 0)

	query := `SELECT d.id, d."name", d."language", d.description, COUNT(c.id) AS total_cards, COUNT(c.id) FILTER(WHERE c.due_at <= now()) AS due_cards, d.created_at
				FROM decks d
				LEFT JOIN cards c ON d.id = c.deck_id
				GROUP BY d.id
				ORDER BY d.created_at DESC, d.id DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get decks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var deck domain.Deck

		if err := rows.Scan(&deck.ID, &deck.Name, &deck.Language, &deck.Description, &deck.TotalCards, &deck.DueCards, &deck.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan decks: %w", err)
		}

		deck.CreatedAt = deck.CreatedAt.UTC()

		decks = append(decks, deck)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decks: %w", err)
	}

	return decks, nil
}

func (r *DeckRepository) GetDeckByID(ctx context.Context, deckID int64) (domain.Deck, error) {
	var deck domain.Deck

	query := `SELECT id, name, language, description, created_at FROM decks WHERE id = $1`
	err := r.pool.QueryRow(ctx, query, deckID).Scan(&deck.ID, &deck.Name, &deck.Language, &deck.Description, &deck.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deck{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Deck{}, fmt.Errorf("get decks by id: %w", err)
	}

	deck.CreatedAt = deck.CreatedAt.UTC()
	return deck, nil
}
