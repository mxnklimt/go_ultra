package domain

import "time"

type Player struct {
	ID        int64
	Username  string
	Rating    int
	CreatedAt time.Time
}

type Stats struct {
	Wins          int
	Losses        int
	WinRate       float64
	CurrentStreak int
	LongestStreak int
}

type Match struct {
	ID                 int64
	WinnerID           int64
	LoserID            int64
	SubmitterID        int64
	WinnerRatingBefore int
	LoserRatingBefore  int
	WinnerRatingAfter  int
	LoserRatingAfter   int
	WinnerDelta        int
	LoserDelta         int
	PlayedAt           time.Time
	CreatedAt          time.Time
	DeletedAt          *time.Time
	DeletedBy          *int64
}
