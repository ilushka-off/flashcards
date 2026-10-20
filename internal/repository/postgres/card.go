package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ilushka-off/flashcards/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CardRepository struct {
	pool *pgxpool.Pool
}

func NewCardRepository(pool *pgxpool.Pool) *CardRepository {
	return &CardRepository{pool: pool}
}

// CreateCard вставляет карточку в колоду deckID, если колода принадлежит
// userID. Владелец проверяется в том же запросе: для чужой или несуществующей
// колоды SELECT не вернёт строк, и вставки не будет — это ErrNotFound без гонки
// между проверкой и вставкой.
//
// Начальное расписание (ease_factor, interval_days, repetitions, created_at,
// updated_at) берётся из DEFAULT схемы, due_at = now() — по тем же часам БД,
// по которым считается due_cards.
func (r *CardRepository) CreateCard(ctx context.Context, deckID, userID int64, in domain.CardCreate) (domain.Card, error) {
	query := `INSERT INTO cards (deck_id, front, back, example, due_at)
				SELECT d.id, $3, $4, $5, now()
				FROM decks d
				WHERE d.id = $1 AND d.user_id = $2
				RETURNING id, deck_id, front, back, example, ease_factor, interval_days, repetitions, due_at, created_at, updated_at`

	var card domain.Card
	err := r.pool.QueryRow(ctx, query, deckID, userID, in.Front, in.Back, in.Example).
		Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Example,
			&card.EaseFactor, &card.IntervalDays, &card.Repetitions,
			&card.DueAt, &card.CreatedAt, &card.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Card{}, fmt.Errorf("create card in deck %d: %w", deckID, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Card{}, fmt.Errorf("create card: %w", err)
	}

	card.DueAt = card.DueAt.UTC()
	card.CreatedAt = card.CreatedAt.UTC()
	card.UpdatedAt = card.UpdatedAt.UTC()
	return card, nil
}
