package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestNewRunsMigrationsAndCreatesTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) error: %v", dbPath, err)
	}
	defer database.Close()

	wantTables := []string{"players", "matches", "sessions", "admin_sessions", "settings"}
	for _, name := range wantTables {
		var got string
		err := database.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&got)
		if err != nil {
			t.Fatalf("table %q not found after migration: %v", name, err)
		}
		if got != name {
			t.Fatalf("expected table %q, got %q", name, got)
		}
	}
}

func TestNewSetsForeignKeysPragma(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer database.Close()

	var fk int
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("query PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys pragma = %d, want 1", fk)
	}

	var mode string
	if err := database.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

func TestNewEnforcesSelfMatchCheck(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(
		`INSERT INTO players (username, rating) VALUES ('alice', 1500)`,
	); err != nil {
		t.Fatalf("insert player: %v", err)
	}

	_, err = insertSelfMatch(database)
	if err == nil {
		t.Fatalf("expected CHECK(winner_id != loser_id) to reject self-match, got nil error")
	}
}

func insertSelfMatch(database *sql.DB) (sql.Result, error) {
	return database.Exec(`
		INSERT INTO matches (
			winner_id, loser_id, submitter_id,
			winner_rating_before, loser_rating_before,
			winner_rating_after, loser_rating_after,
			winner_delta, loser_delta,
			played_at, created_at
		) VALUES (
			1, 1, 1,
			1500, 1500,
			1508, 1492,
			8, -8,
			'2026-06-25T00:00:00Z', '2026-06-25T00:00:00Z'
		)`)
}

func TestNewEnforcesRatingArithmeticCheck(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	database, err := New(dbPath)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(
		`INSERT INTO players (username, rating) VALUES ('alice', 1500)`,
	); err != nil {
		t.Fatalf("insert alice: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO players (username, rating) VALUES ('bob', 1500)`,
	); err != nil {
		t.Fatalf("insert bob: %v", err)
	}

	// winner_rating_after (9999) != winner_rating_before(1500) + winner_delta(8) -> CHECK must reject
	_, err = database.Exec(`
		INSERT INTO matches (
			winner_id, loser_id, submitter_id,
			winner_rating_before, loser_rating_before,
			winner_rating_after, loser_rating_after,
			winner_delta, loser_delta,
			played_at, created_at
		) VALUES (
			1, 2, 1,
			1500, 1500,
			9999, 1492,
			8, -8,
			'2026-06-25T00:00:00Z', '2026-06-25T00:00:00Z'
		)`)
	if err == nil {
		t.Fatalf("expected CHECK(winner_rating_after = winner_rating_before + winner_delta) to reject inconsistent row, got nil error")
	}
}
