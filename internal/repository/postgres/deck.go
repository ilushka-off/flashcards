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

func (r *DeckRepository) CreateDeck(ctx context.Context, in domain.DeckCreate, userID int64) (domain.Deck, error) {
	var deckID int64
	var createdAt time.Time

	query := `INSERT INTO decks(name, language, description, user_id) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, in.Name, in.Language, in.Description, userID).Scan(&deckID, &createdAt)
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

func (r *DeckRepository) GetDeckList(ctx context.Context, userID int64) ([]domain.Deck, error) {
	decks := make([]domain.Deck, 0)

	query := `SELECT d.id, d."name", d."language", d.description, COUNT(c.id) AS total_cards, COUNT(c.id) FILTER(WHERE c.due_at <= now()) AS due_cards, d.created_at
				FROM decks d
				LEFT JOIN cards c ON d.id = c.deck_id
				WHERE d.user_id = $1
				GROUP BY d.id, d."name", d."language", d.description
				ORDER BY d.created_at DESC, d.id DESC`

	rows, err := r.pool.Query(ctx, query, userID)
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

func (r *DeckRepository) GetDeckByID(ctx context.Context, deckID int64, userID int64) (domain.Deck, error) {
	var deck domain.Deck

	query := `SELECT d.id, d."name", d."language", d.description, COUNT(c.id) AS total_cards, COUNT(c.id) FILTER(WHERE c.due_at <= now()) AS due_cards, d.created_at
				FROM decks d
				LEFT JOIN cards c ON d.id = c.deck_id
				WHERE d.id = $1 AND d.user_id = $2
				GROUP BY d.id, d."name", d."language", d.description
				ORDER BY d.created_at DESC, d.id DESC`

	err := r.pool.QueryRow(ctx, query, deckID, userID).Scan(&deck.ID, &deck.Name, &deck.Language, &deck.Description, &deck.TotalCards, &deck.DueCards, &deck.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deck{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Deck{}, fmt.Errorf("get deck by id: %w", err)
	}

	deck.CreatedAt = deck.CreatedAt.UTC()
	return deck, nil
}

func (r *DeckRepository) UpdateDeckByID(ctx context.Context, deckID, userID int64, in domain.DeckUpdate) (domain.Deck, error) {
	var deck domain.Deck

	query := `WITH d AS (
					UPDATE decks SET name = COALESCE($1, name), description = COALESCE($2, description)
					WHERE id = $3 AND user_id = $4
					RETURNING id, "name", "language", description, created_at
				)
				SELECT d.id, d."name", d."language", d.description, COUNT(c.id) AS total_cards, COUNT(c.id) FILTER(WHERE c.due_at <= now()) AS due_cards, d.created_at
				FROM d
				LEFT JOIN cards c ON d.id = c.deck_id
				GROUP BY d.id, d."name", d."language", d.description, d.created_at`

	err := r.pool.QueryRow(ctx, query, in.Name, in.Description, deckID, userID).Scan(&deck.ID, &deck.Name, &deck.Language, &deck.Description, &deck.TotalCards, &deck.DueCards, &deck.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Deck{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Deck{}, fmt.Errorf("update deck: %w", err)
	}

	deck.CreatedAt = deck.CreatedAt.UTC()
	return deck, nil
}

func (r *DeckRepository) DeleteDeckByID(ctx context.Context, deckID, userID int64) error {
	query := `DELETE FROM decks WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, deckID, userID)
	if err != nil {
		return fmt.Errorf("delete deck: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
