package db

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"sync"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var migrationsOnce sync.Once

// New 打开 SQLite 数据库，设置 PRAGMA，运行 goose 迁移，返回 *sql.DB。
func New(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", url.PathEscape(path))
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := runMigrations(database); err != nil {
		database.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return database, nil
}

func runMigrations(database *sql.DB) error {
	var setupErr error
	migrationsOnce.Do(func() {
		goose.SetBaseFS(migrationsFS)
		setupErr = goose.SetDialect("sqlite3")
	})
	if setupErr != nil {
		return fmt.Errorf("set goose dialect: %w", setupErr)
	}
	if err := goose.Up(database, "migrations"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
