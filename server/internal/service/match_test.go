package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go_ultra/internal/domain"
)

func TestMatchService_Record_WinnerIsSubmitterOnWin(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	_, _ = psvc.LoginOrCreate(ctx, "bob")

	res, err := msvc.Record(ctx, alice.ID, "bob", "win", time.Now().UTC())
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}
	if res.MatchID <= 0 {
		t.Fatalf("expected positive match id, got %d", res.MatchID)
	}
	// 单局零和
	if res.WinnerDelta+res.LoserDelta != 0 {
		t.Fatalf("not zero-sum: winner=%d loser=%d", res.WinnerDelta, res.LoserDelta)
	}
	// 平分对局：delta 应为 round(16*0.5)=8
	if res.WinnerDelta != 8 {
		t.Fatalf("winner delta = %d, want 8 (equal ratings)", res.WinnerDelta)
	}
	if res.NewSelfRating != domain.DefaultRating+res.WinnerDelta {
		t.Fatalf("self rating = %d, want %d", res.NewSelfRating, domain.DefaultRating+res.WinnerDelta)
	}
	if res.NewOpponentRating != domain.DefaultRating+res.LoserDelta {
		t.Fatalf("opponent rating = %d, want %d", res.NewOpponentRating, domain.DefaultRating+res.LoserDelta)
	}
}

func TestMatchService_Record_LossSwapsWinner(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	// alice 提交一场 "loss"：winner 应该是 bob，alice 掉分
	res, err := msvc.Record(ctx, alice.ID, "bob", "loss", time.Now().UTC())
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}
	if res.NewSelfRating >= domain.DefaultRating {
		t.Fatalf("submitter (loser) should lose rating, got %d", res.NewSelfRating)
	}
	if res.NewOpponentRating <= domain.DefaultRating {
		t.Fatalf("opponent (winner) should gain rating, got %d", res.NewOpponentRating)
	}
	if res.WinnerDelta+res.LoserDelta != 0 {
		t.Fatalf("not zero-sum")
	}
}

func TestMatchService_Record_SelfMatch(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	_, err := msvc.Record(ctx, alice.ID, "alice", "win", time.Now().UTC())
	if !errors.Is(err, domain.ErrSelfMatch) {
		t.Fatalf("expected ErrSelfMatch, got %v", err)
	}
}

func TestMatchService_Record_OpponentNotFound(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	_, err := msvc.Record(ctx, alice.ID, "ghost", "win", time.Now().UTC())
	if !errors.Is(err, domain.ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}

func TestMatchService_Record_SumConservedOver100Games(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	bob, _ := psvc.LoginOrCreate(ctx, "bob")

	const initialSum = 2 * domain.DefaultRating
	base := time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		result := "win"
		if i%2 == 1 {
			result = "loss" // 交替胜负，分数来回波动
		}
		_, err := msvc.Record(ctx, alice.ID, "bob", result, base.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Record #%d error: %v", i, err)
		}
	}
	// 录入 100 局后，两人 rating 之和必须守恒
	pa, err := psvc.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByUsername alice: %v", err)
	}
	pb, err := psvc.GetByUsername(ctx, "bob")
	if err != nil {
		t.Fatalf("GetByUsername bob: %v", err)
	}
	if pa.Rating+pb.Rating != initialSum {
		t.Fatalf("sum not conserved: %d + %d = %d, want %d", pa.Rating, pb.Rating, pa.Rating+pb.Rating, initialSum)
	}
	_ = alice
	_ = bob
}

func TestMatchService_ListGlobal_And_ListByPlayer_View(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	if _, err := msvc.Record(ctx, alice.ID, "bob", "win", at); err != nil {
		t.Fatalf("Record error: %v", err)
	}

	// 全局
	global, err := msvc.ListGlobal(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListGlobal error: %v", err)
	}
	if len(global) != 1 {
		t.Fatalf("expected 1 global match, got %d", len(global))
	}

	// alice 视角：Result 相对 alice = "win"，Opponent = "bob"
	view, err := msvc.ListByPlayer(ctx, alice.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListByPlayer error: %v", err)
	}
	if len(view) != 1 {
		t.Fatalf("expected 1 match for alice, got %d", len(view))
	}
	mv := view[0]
	if mv.Opponent != "bob" {
		t.Fatalf("opponent = %q, want bob", mv.Opponent)
	}
	if mv.Result != "win" {
		t.Fatalf("result = %q, want win", mv.Result)
	}
	if mv.RatingBefore != domain.DefaultRating {
		t.Fatalf("rating before = %d, want %d", mv.RatingBefore, domain.DefaultRating)
	}
	if mv.RatingAfter != mv.RatingBefore+mv.Delta {
		t.Fatalf("rating math broken: before=%d after=%d delta=%d", mv.RatingBefore, mv.RatingAfter, mv.Delta)
	}
	if mv.Delta <= 0 {
		t.Fatalf("winner delta should be positive, got %d", mv.Delta)
	}
}

func TestMatchService_ListByPlayer_LoserPerspective(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	bob, _ := psvc.LoginOrCreate(ctx, "bob")

	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	msvc.Record(ctx, alice.ID, "bob", "win", at) // alice 赢，bob 输

	view, err := msvc.ListByPlayer(ctx, bob.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListByPlayer error: %v", err)
	}
	mv := view[0]
	if mv.Opponent != "alice" {
		t.Fatalf("opponent = %q, want alice", mv.Opponent)
	}
	if mv.Result != "loss" {
		t.Fatalf("result = %q, want loss", mv.Result)
	}
	if mv.Delta >= 0 {
		t.Fatalf("loser delta should be negative, got %d", mv.Delta)
	}
}

func TestMatchService_History_PrependsStartPoint(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	createdAt := alice.CreatedAt
	at := createdAt.Add(time.Hour)
	msvc.Record(ctx, alice.ID, "bob", "win", at)

	points, err := msvc.History(ctx, alice.ID, createdAt)
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points (start + 1 match), got %d", len(points))
	}
	if !points[0].PlayedAt.Equal(createdAt.UTC()) {
		t.Fatalf("first point time = %v, want %v", points[0].PlayedAt, createdAt.UTC())
	}
	if points[0].Rating != domain.DefaultRating {
		t.Fatalf("first point rating = %d, want %d", points[0].Rating, domain.DefaultRating)
	}
	if points[1].Rating <= domain.DefaultRating {
		t.Fatalf("second point should reflect a win, got %d", points[1].Rating)
	}
}

func TestMatchService_Record_ConcurrentNoBusyErrors(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	const n = 20
	base := time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC)
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result := "win"
			if i%2 == 1 {
				result = "loss"
			}
			_, err := msvc.Record(ctx, alice.ID, "bob", result, base.Add(time.Duration(i)*time.Minute))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Record failed (expected 0 errors with _txlock=immediate): %v", err)
		}
	}
	// 并发录入后两人 rating 之和必须守恒
	pa, _ := psvc.GetByUsername(ctx, "alice")
	pb, _ := psvc.GetByUsername(ctx, "bob")
	if pa.Rating+pb.Rating != 2*domain.DefaultRating {
		t.Fatalf("sum not conserved after concurrent records: %d + %d", pa.Rating, pb.Rating)
	}
}
