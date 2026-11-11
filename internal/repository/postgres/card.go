package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
//
// Все три запроса идут в одной транзакции REPEATABLE READ: они видят один
// снимок данных, поэтому total всегда согласован со страницей. Между запросами
// разных страниц снимки разные — offset-пагинация может пропустить или
// повторить карточку, если колоду меняют во время листания.
func (r *CardRepository) GetCardList(ctx context.Context, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error) {
	var (
		cards []domain.Card
		total int
	)
	txOpts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	err := pgx.BeginTxFunc(ctx, r.pool, txOpts, func(tx pgx.Tx) error {
		var err error
		cards, total, err = getCardList(ctx, tx, deckID, userID, limit, offset, q)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return cards, total, nil
}

// UpdateCardByID меняет только переданные текстовые поля и updated_at.
// Владение карточкой проверяется через колоду в том же запросе.
func (r *CardRepository) UpdateCardByID(ctx context.Context, cardID, userID int64, in domain.CardUpdate, now time.Time) (domain.Card, error) {
	query := `UPDATE cards AS c
		SET front = COALESCE($3, c.front),
			back = COALESCE($4, c.back),
			example = COALESCE($5, c.example),
			updated_at = $6
		FROM decks AS d
		WHERE c.id = $1 AND c.deck_id = d.id AND d.user_id = $2
		RETURNING c.id, c.deck_id, c.front, c.back, c.example, c.ease_factor,
			c.interval_days, c.repetitions, c.due_at, c.created_at, c.updated_at`

	var card domain.Card
	err := r.pool.QueryRow(ctx, query, cardID, userID, in.Front, in.Back, in.Example, now).
		Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Example,
			&card.EaseFactor, &card.IntervalDays, &card.Repetitions,
			&card.DueAt, &card.CreatedAt, &card.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Card{}, fmt.Errorf("update card %d: %w", cardID, domain.ErrNotFound)
	}
	if err != nil {
		return domain.Card{}, fmt.Errorf("update card %d: %w", cardID, err)
	}

	card.DueAt = card.DueAt.UTC()
	card.CreatedAt = card.CreatedAt.UTC()
	card.UpdatedAt = card.UpdatedAt.UTC()
	return card, nil
}

// DeleteByID удаляет карточку пользователя вместе с историей ответов.
// Владение проверяется в том же запросе; reviews удаляются каскадом в БД.
func (r *CardRepository) DeleteByID(ctx context.Context, cardID, userID int64) error {
	query := `DELETE FROM cards AS c
		USING decks AS d
		WHERE c.id = $1 AND c.deck_id = d.id AND d.user_id = $2`
	tag, err := r.pool.Exec(ctx, query, cardID, userID)
	if err != nil {
		return fmt.Errorf("delete card %d: %w", cardID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete card %d: %w", cardID, domain.ErrNotFound)
	}
	return nil
}

func getCardList(ctx context.Context, tx pgx.Tx, deckID, userID int64, limit, offset int, q string) ([]domain.Card, int, error) {
	var exists bool
	err := tx.QueryRow(ctx,
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
	err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM cards c WHERE `+filter,
		deckID, q, pattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count cards: %w", err)
	}

	rows, err := tx.Query(ctx,
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
