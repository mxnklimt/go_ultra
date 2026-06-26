package domain

import "testing"

func TestDan(t *testing.T) {
	tests := []struct {
		rating   float64
		expected int
	}{
		{1049.0, 0},
		{1050.0, 1},
		{1199.0, 1},
		{1200.0, 2},
		{1399.0, 2},
		{1400.0, 3},
		{1500.0, 3},
		{1599.0, 3},
		{1600.0, 4},
		{2399.0, 7},
		{2400.0, 8},
		{2599.0, 8},
		{2600.0, 9},
		{5000.0, 9},
	}
	for _, tt := range tests {
		t.Run(ftoa(tt.rating), func(t *testing.T) {
			got := Dan(tt.rating)
			if got != tt.expected {
				t.Fatalf("Dan(%v) = %d, want %d", tt.rating, got, tt.expected)
			}
		})
	}
}

func TestDanBelowFloorBoundary(t *testing.T) {
	if Dan(RankFloor) != 1 {
		t.Fatalf("Dan(RankFloor=%v) = %d, want 1", RankFloor, Dan(RankFloor))
	}
	if Dan(RankFloor-1) != 0 {
		t.Fatalf("Dan(RankFloor-1=%v) = %d, want 0", RankFloor-1, Dan(RankFloor-1))
	}
}

func TestRankFloorConstant(t *testing.T) {
	if RankFloor != 1050.0 {
		t.Fatalf("RankFloor = %v, want 1050.0", RankFloor)
	}
}

// ftoa は测试内部小工具，避免引入 fmt 仅为子测试命名。
func ftoa(f float64) string {
	return itoa(int(f))
}

// itoa は测试内部小工具，避免引入 strconv 仅为子测试命名。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
