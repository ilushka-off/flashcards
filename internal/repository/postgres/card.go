package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// GetCardList возвращает страницу карточек колоды deckID (новые сверху) и total —
// сколько карточек подходит под фильтр всего. q ищется подстрокой в front или
// back без учёта регистра; пустой q — без фильтра.
//
// Сначала проверяется, что колода есть и принадлежит userID: иначе ErrNotFound,
// чтобы клиент отличал «колоды нет» от «ничего не нашлось». total считается
// отдельным COUNT(*), а не COUNT(*) OVER (): при offset за концом списка
// страница пуста, и total из неё взять было бы неоткуда.
func (r *CardRepository) GetCardList(ctx context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM decks WHERE id = $1 AND user_id = $2)`,
		deckID, userID).Scan(&exists)
	if err != nil {
		return nil, 0, fmt.Errorf("check deck %d: %w", deckID, err)
	}
	if !exists {
		return nil, 0, fmt.Errorf("list cards of deck %d: %w", deckID, domain.ErrNotFound)
	}

	const filter = `c.deck_id = $1
		AND ($2::text = '' OR c.front ILIKE $3 ESCAPE '\' OR c.back ILIKE $3 ESCAPE '\')`
	pattern := "%" + escapeLike(q) + "%"

	var total int
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM cards c WHERE `+filter,
		deckID, q, pattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count cards: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.deck_id, c.front, c.back, c.example, c.ease_factor, c.interval_days,
			c.repetitions, c.due_at, c.created_at, c.updated_at
		FROM cards c
		WHERE `+filter+`
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $4 OFFSET $5`,
		deckID, q, pattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list cards: %w", err)
	}
	defer rows.Close()

	cards := make([]domain.Card, 0)
	for rows.Next() {
		var card domain.Card
		if err := rows.Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Example,
			&card.EaseFactor, &card.IntervalDays, &card.Repetitions,
			&card.DueAt, &card.CreatedAt, &card.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan card: %w", err)
		}
		card.DueAt = card.DueAt.UTC()
		card.CreatedAt = card.CreatedAt.UTC()
		card.UpdatedAt = card.UpdatedAt.UTC()
		cards = append(cards, card)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate cards: %w", err)
	}

	return cards, total, nil
}

// likeEscaper экранирует спецсимволы LIKE, чтобы они искались как обычные
// символы. Обратный слэш идёт первым в паре, и Replacer заменяет всё за один
// проход, поэтому уже добавленные слэши повторно не экранируются.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}
