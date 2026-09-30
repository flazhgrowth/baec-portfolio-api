-- +goose Up
-- +goose StatementBegin
CREATE TYPE sudoku_mode       AS ENUM ('single', 'versus');
CREATE TYPE sudoku_difficulty AS ENUM ('easy', 'medium', 'hard');
CREATE TYPE sudoku_status     AS ENUM ('waiting', 'in_progress', 'completed');
CREATE TYPE sudoku_end_reason AS ENUM ('solved', 'forfeit');

CREATE TABLE sudoku_games (
    id                TEXT PRIMARY KEY,
    mode              sudoku_mode       NOT NULL,
    online            BOOLEAN           NOT NULL DEFAULT FALSE,
    difficulty        sudoku_difficulty NOT NULL,
    status            sudoku_status     NOT NULL,
    join_code         TEXT,
    puzzle            CHAR(81)          NOT NULL,
    solution          CHAR(81)          NOT NULL,
    board             CHAR(81)          NOT NULL,
    rules             JSONB             NOT NULL,
    turn_player_seat  TEXT,
    turn_started_at   TIMESTAMPTZ,
    turn_deadline_at  TIMESTAMPTZ,
    started_at        TIMESTAMPTZ       NOT NULL,
    completed_at      TIMESTAMPTZ,
    end_reason        sudoku_end_reason,
    winner_seat       TEXT,
    version           INTEGER           NOT NULL DEFAULT 1,
    created_by        TEXT              NOT NULL REFERENCES accounts (id),
    created_at        TIMESTAMPTZ       NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ,

    CONSTRAINT sudoku_games_online_only_versus CHECK (NOT online OR mode = 'versus'),
    CONSTRAINT sudoku_games_waiting_only_online CHECK (status <> 'waiting' OR online),
    CONSTRAINT sudoku_games_join_code_iff_waiting CHECK ((join_code IS NOT NULL) = (status = 'waiting')),
    CONSTRAINT sudoku_games_turn_iff_running CHECK (
        (status = 'in_progress' AND mode = 'versus')
        = (turn_player_seat IS NOT NULL AND turn_started_at IS NOT NULL AND turn_deadline_at IS NOT NULL)
    ),
    CONSTRAINT sudoku_games_completed_fields CHECK (
        (status = 'completed') = (completed_at IS NOT NULL AND end_reason IS NOT NULL)
    )
);

-- A join code only has to be unique among open lobbies; it is freed once the game starts.
CREATE UNIQUE INDEX sudoku_games_join_code_uq ON sudoku_games (upper(join_code)) WHERE status = 'waiting';
CREATE INDEX sudoku_games_waiting_idx  ON sudoku_games (created_at)       WHERE status = 'waiting';
CREATE INDEX sudoku_games_deadline_idx ON sudoku_games (turn_deadline_at) WHERE status = 'in_progress';
CREATE INDEX sudoku_games_creator_idx  ON sudoku_games (created_by, created_at DESC);

CREATE TABLE sudoku_game_players (
    game_id               TEXT        NOT NULL REFERENCES sudoku_games (id) ON DELETE CASCADE,
    seat                  TEXT        NOT NULL CHECK (seat IN ('p1', 'p2')),
    user_id               TEXT        REFERENCES accounts (id),
    name                  TEXT        NOT NULL,
    token_hash            TEXT        NOT NULL UNIQUE,
    score                 INTEGER     NOT NULL DEFAULT 0,
    faults                INTEGER     NOT NULL DEFAULT 0,
    mistakes              INTEGER     NOT NULL DEFAULT 0,
    skip_turns_remaining  INTEGER     NOT NULL DEFAULT 0,
    connected             BOOLEAN     NOT NULL DEFAULT TRUE,
    last_seen_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    forfeit_at            TIMESTAMPTZ,
    PRIMARY KEY (game_id, seat)
);
CREATE INDEX sudoku_game_players_user_idx    ON sudoku_game_players (user_id, game_id) WHERE user_id IS NOT NULL;
CREATE INDEX sudoku_game_players_forfeit_idx ON sudoku_game_players (forfeit_at)       WHERE forfeit_at IS NOT NULL;

-- Games and seats reference each other, so both rows are inserted in one transaction.
ALTER TABLE sudoku_games
    ADD CONSTRAINT sudoku_games_turn_fk   FOREIGN KEY (id, turn_player_seat) REFERENCES sudoku_game_players (game_id, seat) DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT sudoku_games_winner_fk FOREIGN KEY (id, winner_seat)      REFERENCES sudoku_game_players (game_id, seat) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE sudoku_moves (
    id            BIGSERIAL   PRIMARY KEY,
    game_id       TEXT        NOT NULL REFERENCES sudoku_games (id) ON DELETE CASCADE,
    seat          TEXT        NOT NULL,
    row           SMALLINT    NOT NULL CHECK (row BETWEEN 0 AND 8),
    col           SMALLINT    NOT NULL CHECK (col BETWEEN 0 AND 8),
    value         SMALLINT    NOT NULL CHECK (value BETWEEN 1 AND 9),
    correct       BOOLEAN     NOT NULL,
    points        INTEGER     NOT NULL DEFAULT 0,
    elapsed_ms    INTEGER,
    game_version  INTEGER     NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sudoku_moves_game_idx ON sudoku_moves (game_id, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sudoku_moves;
ALTER TABLE IF EXISTS sudoku_games DROP CONSTRAINT IF EXISTS sudoku_games_turn_fk, DROP CONSTRAINT IF EXISTS sudoku_games_winner_fk;
DROP TABLE IF EXISTS sudoku_game_players;
DROP TABLE IF EXISTS sudoku_games;
DROP TYPE IF EXISTS sudoku_end_reason;
DROP TYPE IF EXISTS sudoku_status;
DROP TYPE IF EXISTS sudoku_difficulty;
DROP TYPE IF EXISTS sudoku_mode;
-- +goose StatementEnd
