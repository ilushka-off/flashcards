-- Список карточек колоды: GetCardList сортирует по created_at DESC, id DESC.
-- Обратный проход по индексу отдаёт этот порядок без сортировки, и LIMIT
-- останавливает чтение после нужной страницы.
CREATE INDEX cards_deck_id_created_at_idx ON cards (deck_id, created_at, id);
