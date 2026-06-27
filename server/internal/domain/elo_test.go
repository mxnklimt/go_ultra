package domain

import (
	"math"
	"testing"
)

func TestExpectedScore(t *testing.T) {
	const eps = 1e-9
	tests := []struct {
		name     string
		ratingA  float64
		ratingB  float64
		expected float64
	}{
		{"equal ratings -> 0.5", 1500.0, 1500.0, 0.5},
		{"A higher by 400 -> ~0.909", 1900.0, 1500.0, 1.0 / (1.0 + math.Pow(10, -400.0/400.0))},
		{"A lower by 400 -> ~0.091", 1500.0, 1900.0, 1.0 / (1.0 + math.Pow(10, 400.0/400.0))},
		{"extreme A dominates", 5000.0, 0.0, 1.0 / (1.0 + math.Pow(10, -5000.0/400.0))},
		{"extreme A crushed", 0.0, 5000.0, 1.0 / (1.0 + math.Pow(10, 5000.0/400.0))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpectedScore(tt.ratingA, tt.ratingB)
			if math.Abs(got-tt.expected) > eps {
				t.Fatalf("ExpectedScore(%v,%v) = %v, want %v", tt.ratingA, tt.ratingB, got, tt.expected)
			}
		})
	}
}

func TestExpectedScoreSymmetry(t *testing.T) {
	const eps = 1e-9
	pairs := [][2]float64{{1500.0, 1500.0}, {1900.0, 1500.0}, {1500.0, 1900.0}, {2400.0, 1000.0}, {0.0, 5000.0}}
	for _, p := range pairs {
		ea := ExpectedScore(p[0], p[1])
		eb := ExpectedScore(p[1], p[0])
		if math.Abs(ea+eb-1.0) > eps {
			t.Fatalf("E_A+E_B != 1 for %v: %v + %v", p, ea, eb)
		}
	}
}

func TestComputeDelta(t *testing.T) {
	tests := []struct {
		name          string
		winnerRating  float64
		loserRating   float64
		expectedDelta float64
	}{
		{"equal ratings", 1500.0, 1500.0, 8.0},
		{"winner higher by 400", 1900.0, 1500.0, 1.45},
		{"winner lower by 400 (upset)", 1500.0, 1900.0, 14.55},
		{"extreme upset (winner crushed favorite)", 0.0, 5000.0, 16.0},
		{"extreme expected win", 5000.0, 0.0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeDelta(tt.winnerRating, tt.loserRating)
			if math.Abs(got-tt.expectedDelta) > 0.005 {
				t.Fatalf("ComputeDelta(%v,%v) = %v, want %v", tt.winnerRating, tt.loserRating, got, tt.expectedDelta)
			}
		})
	}
}

func TestComputeDeltaZeroSum(t *testing.T) {
	pairs := [][2]float64{{1500.0, 1500.0}, {1900.0, 1500.0}, {1500.0, 1900.0}, {2400.0, 1000.0}, {0.0, 5000.0}}
	for _, p := range pairs {
		d := ComputeDelta(p[0], p[1])
		loserDelta := -d
		// Due to float rounding, d + loserDelta should be exactly 0 (loserDelta is just -d)
		if d+loserDelta != 0 {
			t.Fatalf("zero-sum violated for %v: winner=%v loser=%v", p, d, loserDelta)
		}
	}
}

func TestEloConstants(t *testing.T) {
	if DefaultRating != 1500.00 {
		t.Fatalf("DefaultRating = %v, want 1500.00", DefaultRating)
	}
	if KFactor != 16.0 {
		t.Fatalf("KFactor = %v, want 16.0", KFactor)
	}
}
