package service

import (
	"testing"
	"time"

	"go_ultra/internal/domain"
)

func TestLeaderboardService_List_RankAndStats(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	// alice 赢一局 → alice 升、bob 降
	msvc.Record(ctx, alice.ID, "bob", "win", time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC))

	rows, err := lsvc.List(ctx, 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// rank 1 应为分高者 alice
	if rows[0].Rank != 1 || rows[0].Username != "alice" {
		t.Fatalf("rank 1 should be alice, got rank=%d user=%q", rows[0].Rank, rows[0].Username)
	}
	if rows[1].Rank != 2 || rows[1].Username != "bob" {
		t.Fatalf("rank 2 should be bob, got rank=%d user=%q", rows[1].Rank, rows[1].Username)
	}
	// alice：1 胜 0 负，games=1，winrate=1.0
	if rows[0].GamesPlayed != 1 || rows[0].WinRate != 1.0 {
		t.Fatalf("alice stats wrong: games=%d winrate=%v", rows[0].GamesPlayed, rows[0].WinRate)
	}
	// Dan 必须与 domain.Dan 一致
	if rows[0].Dan != domain.Dan(rows[0].Rating) {
		t.Fatalf("alice dan = %d, want %d", rows[0].Dan, domain.Dan(rows[0].Rating))
	}
}

func TestLeaderboardService_List_MinGamesFilter(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")
	psvc.LoginOrCreate(ctx, "carol") // 0 局

	msvc.Record(ctx, alice.ID, "bob", "win", time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC))

	// min_games=1：carol（0 局）必须被过滤掉
	rows, err := lsvc.List(ctx, 1)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	for _, r := range rows {
		if r.Username == "carol" {
			t.Fatalf("carol (0 games) should be filtered out by min_games=1")
		}
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows with min_games=1, got %d", len(rows))
	}
	// rank 在过滤后仍连续：1,2
	if rows[0].Rank != 1 || rows[1].Rank != 2 {
		t.Fatalf("ranks not contiguous after filter: %d, %d", rows[0].Rank, rows[1].Rank)
	}
}

func TestLeaderboardService_CompareData_SeriesAndHeadToHead(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")
	psvc.LoginOrCreate(ctx, "carol")

	base := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	// alice vs bob：alice 赢 2, bob 赢 1
	msvc.Record(ctx, alice.ID, "bob", "win", base.Add(1*time.Minute))
	msvc.Record(ctx, alice.ID, "bob", "win", base.Add(2*time.Minute))
	msvc.Record(ctx, alice.ID, "bob", "loss", base.Add(3*time.Minute)) // bob 赢

	res, err := lsvc.CompareData(ctx, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("CompareData error: %v", err)
	}
	if len(res.Series) != 3 {
		t.Fatalf("expected 3 series, got %d", len(res.Series))
	}
	// 每条 series 开头都有起点（createdAt, DefaultRating）
	for _, s := range res.Series {
		if len(s.Points) == 0 {
			t.Fatalf("series %q has no points", s.Username)
		}
		if s.Points[0].Rating != domain.DefaultRating {
			t.Fatalf("series %q first point rating = %v, want %v", s.Username, s.Points[0].Rating, domain.DefaultRating)
		}
		if s.Color == "" {
			t.Fatalf("series %q has empty color", s.Username)
		}
	}
	// 5 色板：前 3 条颜色取调色板前 3 个
	wantPalette := []string{"#4a9eff", "#7fd6a3", "#8b5cf6"}
	for i, s := range res.Series {
		if s.Color != wantPalette[i] {
			t.Fatalf("series[%d] color = %q, want %q", i, s.Color, wantPalette[i])
		}
	}
	// head_to_head：C(3,2)=3 对
	if len(res.HeadToHead) != 3 {
		t.Fatalf("expected 3 head-to-head pairs, got %d", len(res.HeadToHead))
	}
	// 找 alice-bob 对：alice 2 胜 bob 1 胜
	var found bool
	for _, h := range res.HeadToHead {
		if (h.A == "alice" && h.B == "bob") || (h.A == "bob" && h.B == "alice") {
			found = true
			if h.A == "alice" {
				if h.AWins != 2 || h.BWins != 1 {
					t.Fatalf("alice-bob h2h = %d-%d, want 2-1", h.AWins, h.BWins)
				}
			} else { // A==bob
				if h.AWins != 1 || h.BWins != 2 {
					t.Fatalf("bob-alice h2h = %d-%d, want 1-2", h.AWins, h.BWins)
				}
			}
		}
	}
	if !found {
		t.Fatalf("alice-bob pair not found in head-to-head")
	}
}

func TestLeaderboardService_CompareData_ColorWrapsAfterFive(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	names := []string{"usr1", "usr2", "usr3", "usr4", "usr5", "usr6"}
	for _, n := range names {
		psvc.LoginOrCreate(ctx, n)
	}
	res, err := lsvc.CompareData(ctx, names)
	if err != nil {
		t.Fatalf("CompareData error: %v", err)
	}
	if len(res.Series) != 6 {
		t.Fatalf("expected 6 series, got %d", len(res.Series))
	}
	// 第 6 条（index 5）颜色应循环回调色板第 1 个
	if res.Series[5].Color != res.Series[0].Color {
		t.Fatalf("color did not wrap: series[5]=%q series[0]=%q", res.Series[5].Color, res.Series[0].Color)
	}
}

func TestLeaderboardService_CompareData_PlayerNotFound(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	psvc.LoginOrCreate(ctx, "alice")
	_, err := lsvc.CompareData(ctx, []string{"alice", "ghost"})
	if err == nil {
		t.Fatalf("expected error for unknown username, got nil")
	}
}
