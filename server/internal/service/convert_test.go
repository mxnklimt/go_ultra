package service

import (
	"database/sql"
	"testing"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

func TestParseTime(t *testing.T) {
	want := time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)
	got, err := parseTime("2026-06-25T14:30:00Z")
	if err != nil {
		t.Fatalf("parseTime error: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("parseTime = %v, want %v", got, want)
	}
	if _, err := parseTime("not-a-time"); err == nil {
		t.Fatalf("parseTime(bad) expected error, got nil")
	}
}

func TestFormatTime(t *testing.T) {
	in := time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)
	if got := formatTime(in); got != "2026-06-25T14:30:00Z" {
		t.Fatalf("formatTime = %q, want %q", got, "2026-06-25T14:30:00Z")
	}
	// 非 UTC 输入必须被规整为 UTC 再格式化
	loc := time.FixedZone("X", 3600)
	in2 := time.Date(2026, 6, 25, 15, 30, 0, 0, loc) // == 14:30Z
	if got := formatTime(in2); got != "2026-06-25T14:30:00Z" {
		t.Fatalf("formatTime(non-utc) = %q, want %q", got, "2026-06-25T14:30:00Z")
	}
}

func TestToDomainPlayer(t *testing.T) {
	row := sqlc.Player{
		ID:        7,
		Username:  "alice",
		Rating:    1500,
		CreatedAt: "2026-06-25T14:30:00Z",
	}
	p, err := toDomainPlayer(row)
	if err != nil {
		t.Fatalf("toDomainPlayer error: %v", err)
	}
	if p.ID != 7 || p.Username != "alice" || p.Rating != 1500 {
		t.Fatalf("toDomainPlayer scalar mismatch: %+v", p)
	}
	if !p.CreatedAt.Equal(time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)) {
		t.Fatalf("toDomainPlayer time mismatch: %v", p.CreatedAt)
	}
}

func TestToDomainMatch(t *testing.T) {
	del := "2026-06-26T00:00:00Z"
	row := sqlc.Match{
		ID:                 3,
		WinnerID:           1,
		LoserID:            2,
		SubmitterID:        1,
		WinnerRatingBefore: 1500,
		LoserRatingBefore:  1500,
		WinnerRatingAfter:  1508,
		LoserRatingAfter:   1492,
		WinnerDelta:        8,
		LoserDelta:         -8,
		PlayedAt:           "2026-06-25T14:30:00Z",
		CreatedAt:          "2026-06-25T14:30:01Z",
		DeletedAt:          sql.NullString{String: del, Valid: true},
		DeletedBy:          sql.NullInt64{Valid: false},
	}
	m, err := toDomainMatch(row)
	if err != nil {
		t.Fatalf("toDomainMatch error: %v", err)
	}
	if m.ID != 3 || m.WinnerID != 1 || m.LoserID != 2 || m.WinnerDelta != 8 || m.LoserDelta != -8 {
		t.Fatalf("toDomainMatch scalar mismatch: %+v", m)
	}
	if m.DeletedAt == nil || !m.DeletedAt.Equal(time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("toDomainMatch DeletedAt mismatch: %v", m.DeletedAt)
	}
	if m.DeletedBy != nil {
		t.Fatalf("toDomainMatch DeletedBy should be nil, got %v", *m.DeletedBy)
	}
}

// 编译期保证 domain 包被使用（避免误删 import）
var _ = domain.DefaultRating
