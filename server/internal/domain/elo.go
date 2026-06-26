package domain

import "math"

// DefaultRating 是新玩家的起始等级分。换场景时改此常量。
const DefaultRating = 1500

// KFactor 控制每局分数变动幅度。换场景时改此常量。
const KFactor = 16

// ExpectedScore 返回 A 对 B 的期望胜率：1/(1+10^((B-A)/400))。
func ExpectedScore(ratingA, ratingB int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(ratingB-ratingA)/400.0))
}

// ComputeDelta 返回胜者应获得的分数变动（正整数或 0），
// 采用 math.Round（half away from zero）。败者变动为其相反数。
func ComputeDelta(winnerRating, loserRating int) int {
	eWinner := ExpectedScore(winnerRating, loserRating)
	return int(math.Round(KFactor * (1.0 - eWinner)))
}
