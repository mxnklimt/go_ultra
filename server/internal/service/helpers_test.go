package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"go_ultra/internal/db"
	"go_ultra/internal/db/sqlc"
)

// newTestDB 在测试临时目录开一个全新的 sqlite 文件，跑迁移，返回 *sql.DB 与 *sqlc.Queries。
// 测试结束自动关闭连接（临时目录由 testing 框架清理）。
func newTestDB(t *testing.T) (*sql.DB, *sqlc.Queries) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	sqlDB, err := db.New(path)
	if err != nil {
		t.Fatalf("db.New(%q) error: %v", path, err)
	}
	t.Cleanup(func() {
		if cerr := sqlDB.Close(); cerr != nil {
			t.Errorf("close db: %v", cerr)
		}
	})
	return sqlDB, sqlc.New(sqlDB)
}

// ctx 返回带超时的测试上下文，超时后自动取消。
func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// mustCreatePlayer 直接经 Queries 建一个玩家，返回其 ID（绕过 service，仅用于布置测试前置数据）。
func mustCreatePlayer(t *testing.T, q *sqlc.Queries, username string, rating float64) int64 {
	t.Helper()
	p, err := q.CreatePlayer(context.Background(), sqlc.CreatePlayerParams{
		Username: username,
		Rating:   rating,
	})
	if err != nil {
		t.Fatalf("CreatePlayer(%q) error: %v", username, err)
	}
	return p.ID
}

// TestScaffold 自检：确认临时库能开、迁移能跑、能建玩家。
func TestScaffold(t *testing.T) {
	sqlDB, queries := newTestDB(t)
	if sqlDB == nil || queries == nil {
		t.Fatal("newTestDB returned nil")
	}
	id := mustCreatePlayer(t, queries, "scaffold_user", 1500.00)
	if id <= 0 {
		t.Fatalf("expected positive player id, got %d", id)
	}
	got, err := queries.GetPlayerByID(testCtx(t), id)
	if err != nil {
		t.Fatalf("GetPlayerByID error: %v", err)
	}
	if got.Username != "scaffold_user" || got.Rating != 1500.00 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
