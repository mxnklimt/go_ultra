-- +goose Up
-- +goose StatementBegin
CREATE TABLE players (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE COLLATE NOCASE,
    rating          REAL NOT NULL DEFAULT 1500.00,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_players_rating ON players(rating DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE matches (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    winner_id       INTEGER NOT NULL REFERENCES players(id),
    loser_id        INTEGER NOT NULL REFERENCES players(id),
    submitter_id    INTEGER NOT NULL REFERENCES players(id),

    winner_rating_before  REAL NOT NULL,
    loser_rating_before   REAL NOT NULL,
    winner_rating_after   REAL NOT NULL,
    loser_rating_after    REAL NOT NULL,
    winner_delta          REAL NOT NULL,
    loser_delta           REAL NOT NULL,

    played_at       TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

    deleted_at      TEXT,
    deleted_by      INTEGER REFERENCES players(id),

    CHECK (winner_id != loser_id),
    CHECK (ABS(winner_rating_after - (winner_rating_before + winner_delta)) < 0.001),
    CHECK (ABS(loser_rating_after  - (loser_rating_before  + loser_delta)) < 0.001),
    CHECK (ABS(winner_delta + loser_delta) < 0.001)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_matches_winner ON matches(winner_id, played_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_matches_loser ON matches(loser_id, played_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_matches_played ON matches(played_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_matches_active ON matches(deleted_at) WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,
    player_id   INTEGER NOT NULL REFERENCES players(id),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_sessions_player ON sessions(player_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE admin_sessions (
    token       TEXT PRIMARY KEY,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS settings;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS admin_sessions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS sessions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS matches;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS players;
-- +goose StatementEnd
