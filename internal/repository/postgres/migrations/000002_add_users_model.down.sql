DROP INDEX IF EXISTS decks_user_id_idx;

ALTER TABLE decks DROP COLUMN IF EXISTS user_id;

DROP INDEX IF EXISTS users_user_email_idx;

DROP TABLE IF EXISTS users;
