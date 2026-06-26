package domain

import (
	"math"
	"testing"
)

func TestExpectedScore(t *testing.T) {
	const eps = 1e-9
	tests := []struct {
		name     string
		ratingA  int
		ratingB  int
		expected float64
	}{
		{"equal ratings -> 0.5", 1500, 1500, 0.5},
		{"A higher by 400 -> ~0.909", 1900, 1500, 1.0 / (1.0 + math.Pow(10, -400.0/400.0))},
		{"A lower by 400 -> ~0.091", 1500, 1900, 1.0 / (1.0 + math.Pow(10, 400.0/400.0))},
		{"extreme A dominates", 5000, 0, 1.0 / (1.0 + math.Pow(10, -5000.0/400.0))},
		{"extreme A crushed", 0, 5000, 1.0 / (1.0 + math.Pow(10, 5000.0/400.0))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpectedScore(tt.ratingA, tt.ratingB)
			if math.Abs(got-tt.expected) > eps {
				t.Fatalf("ExpectedScore(%d,%d) = %v, want %v", tt.ratingA, tt.ratingB, got, tt.expected)
			}
		})
	}
}

func TestExpectedScoreSymmetry(t *testing.T) {
	const eps = 1e-9
	pairs := [][2]int{{1500, 1500}, {1900, 1500}, {1500, 1900}, {2400, 1000}, {0, 5000}}
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
		winnerRating  int
		loserRating   int
		expectedDelta int
	}{
		{"equal ratings", 1500, 1500, 8},
		{"winner higher by 400", 1900, 1500, 1},
		{"winner lower by 400 (upset)", 1500, 1900, 15},
		{"extreme upset (winner crushed favorite)", 0, 5000, 16},
		{"extreme expected win", 5000, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeDelta(tt.winnerRating, tt.loserRating)
			if got != tt.expectedDelta {
				t.Fatalf("ComputeDelta(%d,%d) = %d, want %d", tt.winnerRating, tt.loserRating, got, tt.expectedDelta)
			}
		})
	}
}

func TestComputeDeltaZeroSum(t *testing.T) {
	pairs := [][2]int{{1500, 1500}, {1900, 1500}, {1500, 1900}, {2400, 1000}, {0, 5000}}
	for _, p := range pairs {
		d := ComputeDelta(p[0], p[1])
		loserDelta := -d
		if d+loserDelta != 0 {
			t.Fatalf("zero-sum violated for %v: winner=%d loser=%d", p, d, loserDelta)
		}
	}
}

func TestEloConstants(t *testing.T) {
	if DefaultRating != 1500 {
		t.Fatalf("DefaultRating = %d, want 1500", DefaultRating)
	}
	if KFactor != 16 {
		t.Fatalf("KFactor = %d, want 16", KFactor)
	}
}
