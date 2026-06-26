package domain

import "time"

type Player struct {
	ID        int64
	Username  string
	Rating    float64
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
	WinnerRatingBefore float64
	LoserRatingBefore  float64
	WinnerRatingAfter  float64
	LoserRatingAfter   float64
	WinnerDelta        float64
	LoserDelta         float64
	PlayedAt           time.Time
	CreatedAt          time.Time
	DeletedAt          *time.Time
	DeletedBy          *int64
}
