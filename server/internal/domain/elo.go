package domain

import "math"

// DefaultRating は新玩家的起始等级分。换场景时改此常量。
const DefaultRating = 1500.00

// KFactor 控制每局分数变动幅度。换场景时改此常量。
const KFactor = 16.0

// ExpectedScore 返回 A 对 B 的期望胜率：1/(1+10^((B-A)/400))。
func ExpectedScore(ratingA, ratingB float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, (ratingB-ratingA)/400.0))
}

// round2 四舍五入到小数点后两位。
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ComputeDelta 返回胜者应获得的分数变动（精确到小数点后两位），
// 败者变动为其相反数。
func ComputeDelta(winnerRating, loserRating float64) float64 {
	eWinner := ExpectedScore(winnerRating, loserRating)
	return round2(KFactor * (1.0 - eWinner))
}
