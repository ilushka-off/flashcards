CREATE TABLE decks (
                       id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                       name        VARCHAR(100) NOT NULL CHECK (btrim(name) <> ''),
                       language    CHAR(2)      NOT NULL CHECK (language IN ('en', 'es')),
                       description VARCHAR(500) NOT NULL DEFAULT '',
                       created_at  TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE cards (
                       id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                       deck_id       BIGINT       NOT NULL REFERENCES decks (id) ON DELETE CASCADE,
                       front         VARCHAR(200) NOT NULL CHECK (btrim(front) <> ''),
                       back          VARCHAR(200) NOT NULL CHECK (btrim(back) <> ''),
                       example       VARCHAR(500) NOT NULL DEFAULT '',
                       ease_factor   NUMERIC(6,2) NOT NULL DEFAULT 2.50 CHECK (ease_factor >= 1.30),
                       interval_days INTEGER      NOT NULL DEFAULT 0    CHECK (interval_days >= 0),
                       repetitions   INTEGER      NOT NULL DEFAULT 0    CHECK (repetitions >= 0),
                       due_at        TIMESTAMPTZ  NOT NULL,
                       created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
                       updated_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE reviews (
                         id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
                         card_id       BIGINT       NOT NULL REFERENCES cards (id) ON DELETE CASCADE,
                         grade         VARCHAR(5)   NOT NULL CHECK (grade IN ('again', 'hard', 'good', 'easy')),
                         reviewed_at   TIMESTAMPTZ  NOT NULL,
                         interval_days INTEGER      NOT NULL CHECK (interval_days >= 1),
                         ease_factor   NUMERIC(6,2) NOT NULL CHECK (ease_factor >= 1.30)
);

CREATE INDEX cards_deck_id_due_at_idx ON cards (deck_id, due_at);

CREATE INDEX reviews_card_id_reviewed_at_idx ON reviews (card_id, reviewed_at);