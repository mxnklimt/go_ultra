package domain

import "testing"

func TestDan(t *testing.T) {
	tests := []struct {
		rating   int
		expected int
	}{
		{1049, 0},
		{1050, 1},
		{1199, 1},
		{1200, 2},
		{1399, 2},
		{1400, 3},
		{1500, 3},
		{1599, 3},
		{1600, 4},
		{2399, 7},
		{2400, 8},
		{2599, 8},
		{2600, 9},
		{5000, 9},
	}
	for _, tt := range tests {
		t.Run(itoa(tt.rating), func(t *testing.T) {
			got := Dan(tt.rating)
			if got != tt.expected {
				t.Fatalf("Dan(%d) = %d, want %d", tt.rating, got, tt.expected)
			}
		})
	}
}

func TestDanBelowFloorBoundary(t *testing.T) {
	if Dan(RankFloor) != 1 {
		t.Fatalf("Dan(RankFloor=%d) = %d, want 1", RankFloor, Dan(RankFloor))
	}
	if Dan(RankFloor-1) != 0 {
		t.Fatalf("Dan(RankFloor-1=%d) = %d, want 0", RankFloor-1, Dan(RankFloor-1))
	}
}

func TestRankFloorConstant(t *testing.T) {
	if RankFloor != 1050 {
		t.Fatalf("RankFloor = %d, want 1050", RankFloor)
	}
}

// itoa 是测试内部小工具，避免引入 strconv 仅为子测试命名。
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
