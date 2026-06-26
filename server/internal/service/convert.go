package service

import (
	"database/sql"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

// parseTime 把 RFC3339 字符串解析为 UTC time.Time。
func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// formatTime 把 time.Time 规整为 UTC 后格式化为 RFC3339 字符串。
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// nullStringTimePtr 把可空时间列转换为 *time.Time。
func nullStringTimePtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// nullInt64Ptr 把可空整型列转换为 *int64。
func nullInt64Ptr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	v := ni.Int64
	return &v
}

// toDomainPlayer 把 sqlc.Player 行映射为 domain.Player。
func toDomainPlayer(p sqlc.Player) (domain.Player, error) {
	createdAt, err := parseTime(p.CreatedAt)
	if err != nil {
		return domain.Player{}, err
	}
	return domain.Player{
		ID:        p.ID,
		Username:  p.Username,
		Rating:    int(p.Rating),
		CreatedAt: createdAt,
	}, nil
}

// toDomainMatch 把 sqlc.Match 行映射为 domain.Match。
func toDomainMatch(m sqlc.Match) (domain.Match, error) {
	playedAt, err := parseTime(m.PlayedAt)
	if err != nil {
		return domain.Match{}, err
	}
	createdAt, err := parseTime(m.CreatedAt)
	if err != nil {
		return domain.Match{}, err
	}
	deletedAt, err := nullStringTimePtr(m.DeletedAt)
	if err != nil {
		return domain.Match{}, err
	}
	return domain.Match{
		ID:                 m.ID,
		WinnerID:           m.WinnerID,
		LoserID:            m.LoserID,
		SubmitterID:        m.SubmitterID,
		WinnerRatingBefore: int(m.WinnerRatingBefore),
		LoserRatingBefore:  int(m.LoserRatingBefore),
		WinnerRatingAfter:  int(m.WinnerRatingAfter),
		LoserRatingAfter:   int(m.LoserRatingAfter),
		WinnerDelta:        int(m.WinnerDelta),
		LoserDelta:         int(m.LoserDelta),
		PlayedAt:           playedAt,
		CreatedAt:          createdAt,
		DeletedAt:          deletedAt,
		DeletedBy:          nullInt64Ptr(m.DeletedBy),
	}, nil
}
