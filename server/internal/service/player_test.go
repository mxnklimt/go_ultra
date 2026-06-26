package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

func TestPlayerService_LoginOrCreate_CreatesWithDefaultRating(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	p, err := svc.LoginOrCreate(ctx, "  Alice  ") // 含空白，必须 trim
	if err != nil {
		t.Fatalf("LoginOrCreate error: %v", err)
	}
	if p.Username != "Alice" {
		t.Fatalf("username not trimmed: %q", p.Username)
	}
	if p.Rating != domain.DefaultRating {
		t.Fatalf("rating = %v, want %v", p.Rating, domain.DefaultRating)
	}
	if p.ID <= 0 {
		t.Fatalf("expected positive id, got %d", p.ID)
	}
}

func TestPlayerService_LoginOrCreate_Idempotent(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	first, err := svc.LoginOrCreate(ctx, "bob")
	if err != nil {
		t.Fatalf("first LoginOrCreate error: %v", err)
	}
	// 第二次（大小写不同，依赖 username 列 COLLATE NOCASE）必须复用同一行
	second, err := svc.LoginOrCreate(ctx, "BOB")
	if err != nil {
		t.Fatalf("second LoginOrCreate error: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("not idempotent: first id=%d second id=%d", first.ID, second.ID)
	}
	all, err := svc.ListByRating(ctx)
	if err != nil {
		t.Fatalf("ListByRating error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 player, got %d", len(all))
	}
}

func TestPlayerService_LoginOrCreate_Validation(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	cases := []string{
		"",                       // 空
		"   ",                    // 全空白 trim 后为空
		"ab",                     // 2 字符，太短
		string(make([]byte, 33)), // 33 字节，太长（占位长度）
	}
	for _, name := range cases {
		if _, err := svc.LoginOrCreate(ctx, name); err == nil {
			t.Fatalf("LoginOrCreate(%q) expected error, got nil", name)
		}
	}
	// 边界：恰好 3 与恰好 32 必须成功
	if _, err := svc.LoginOrCreate(ctx, "abc"); err != nil {
		t.Fatalf("LoginOrCreate(3 chars) error: %v", err)
	}
	name32 := "abcdefghijklmnopqrstuvwxyz012345" // 32 字符
	if len([]rune(name32)) != 32 {
		t.Fatalf("test fixture wrong length: %d", len([]rune(name32)))
	}
	if _, err := svc.LoginOrCreate(ctx, name32); err != nil {
		t.Fatalf("LoginOrCreate(32 chars) error: %v", err)
	}
}

func TestPlayerService_GetByUsername(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	created, err := svc.LoginOrCreate(ctx, "carol")
	if err != nil {
		t.Fatalf("LoginOrCreate error: %v", err)
	}
	got, err := svc.GetByUsername(ctx, "carol")
	if err != nil {
		t.Fatalf("GetByUsername error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id mismatch: %d vs %d", got.ID, created.ID)
	}
	// 不存在 → ErrPlayerNotFound
	_, err = svc.GetByUsername(ctx, "nobody")
	if !errors.Is(err, domain.ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}

// insertMatch 直接经 Queries 录入一条对局（绕过 service，用于精确布置 streak 数据）。
func insertMatch(t *testing.T, q *sqlc.Queries, winnerID, loserID, submitterID int64, playedAt time.Time) {
	t.Helper()
	_, err := q.CreateMatch(context.Background(), sqlc.CreateMatchParams{
		WinnerID:           winnerID,
		LoserID:            loserID,
		SubmitterID:        submitterID,
		WinnerRatingBefore: 1500.00,
		LoserRatingBefore:  1500.00,
		WinnerRatingAfter:  1508.00,
		LoserRatingAfter:   1492.00,
		WinnerDelta:        8.00,
		LoserDelta:         -8.00,
		PlayedAt:           formatTime(playedAt),
		CreatedAt:          formatTime(playedAt),
	})
	if err != nil {
		t.Fatalf("CreateMatch error: %v", err)
	}
}

func TestPlayerService_GetStats_StreaksAndWinRate(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	a := mustCreatePlayer(t, q, "streak_a", 1500.00)
	b := mustCreatePlayer(t, q, "streak_b", 1500.00)

	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	// 时间升序录入：a 的结果序列为 W W L W W W（最后三连胜）
	// played_at 递增，确保历史遍历顺序确定
	insertMatch(t, q, a, b, a, base.Add(1*time.Minute)) // a win
	insertMatch(t, q, a, b, a, base.Add(2*time.Minute)) // a win
	insertMatch(t, q, b, a, b, base.Add(3*time.Minute)) // a loss
	insertMatch(t, q, a, b, a, base.Add(4*time.Minute)) // a win
	insertMatch(t, q, a, b, a, base.Add(5*time.Minute)) // a win
	insertMatch(t, q, a, b, a, base.Add(6*time.Minute)) // a win

	stats, err := svc.GetStats(ctx, a)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if stats.Wins != 5 || stats.Losses != 1 {
		t.Fatalf("wins/losses = %d/%d, want 5/1", stats.Wins, stats.Losses)
	}
	wantRate := 5.0 / 6.0
	if stats.WinRate < wantRate-1e-9 || stats.WinRate > wantRate+1e-9 {
		t.Fatalf("win rate = %v, want %v", stats.WinRate, wantRate)
	}
	if stats.CurrentStreak != 3 {
		t.Fatalf("current streak = %d, want 3", stats.CurrentStreak)
	}
	if stats.LongestStreak != 3 {
		t.Fatalf("longest streak = %d, want 3", stats.LongestStreak)
	}
}

func TestPlayerService_GetStats_NoMatches(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	a := mustCreatePlayer(t, q, "lonely", 1500.00)
	stats, err := svc.GetStats(ctx, a)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if stats.Wins != 0 || stats.Losses != 0 || stats.WinRate != 0 ||
		stats.CurrentStreak != 0 || stats.LongestStreak != 0 {
		t.Fatalf("expected all-zero stats, got %+v", stats)
	}
}

func TestPlayerService_ListByRating_Ordered(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	mustCreatePlayer(t, q, "low", 1400.00)
	mustCreatePlayer(t, q, "high", 1600.00)
	mustCreatePlayer(t, q, "mid", 1500.00)

	list, err := svc.ListByRating(ctx)
	if err != nil {
		t.Fatalf("ListByRating error: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 players, got %d", len(list))
	}
	if list[0].Username != "high" || list[1].Username != "mid" || list[2].Username != "low" {
		t.Fatalf("not sorted by rating desc: %v", []string{list[0].Username, list[1].Username, list[2].Username})
	}
}
