package domain

import (
	"testing"
	"time"
)

func TestPlayerFields(t *testing.T) {
	now := time.Now().UTC()
	p := Player{ID: 1, Username: "alice", Rating: 1500.00, CreatedAt: now}
	if p.ID != 1 || p.Username != "alice" || p.Rating != 1500.00 || !p.CreatedAt.Equal(now) {
		t.Fatalf("Player fields not assignable as expected: %+v", p)
	}
}

func TestStatsFields(t *testing.T) {
	s := Stats{Wins: 3, Losses: 2, WinRate: 0.6, CurrentStreak: 1, LongestStreak: 4}
	if s.Wins != 3 || s.Losses != 2 || s.WinRate != 0.6 || s.CurrentStreak != 1 || s.LongestStreak != 4 {
		t.Fatalf("Stats fields not assignable as expected: %+v", s)
	}
}

func TestMatchFields(t *testing.T) {
	now := time.Now().UTC()
	delBy := int64(7)
	m := Match{
		ID:                 10,
		WinnerID:           1,
		LoserID:            2,
		SubmitterID:        1,
		WinnerRatingBefore: 1500.00,
		LoserRatingBefore:  1500.00,
		WinnerRatingAfter:  1508.00,
		LoserRatingAfter:   1492.00,
		WinnerDelta:        8.00,
		LoserDelta:         -8.00,
		PlayedAt:           now,
		CreatedAt:          now,
		DeletedAt:          &now,
		DeletedBy:          &delBy,
	}
	if m.WinnerDelta+m.LoserDelta != 0 {
		t.Fatalf("expected zero-sum deltas, got %v and %v", m.WinnerDelta, m.LoserDelta)
	}
	if m.DeletedAt == nil || *m.DeletedBy != 7 {
		t.Fatalf("pointer fields not assignable as expected: %+v", m)
	}
}
