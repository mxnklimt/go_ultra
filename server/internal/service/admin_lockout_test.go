package service

import (
	"errors"
	"testing"
	"time"

	"go_ultra/internal/domain"
)

// newLockoutTestService 构造一个仅用于退避逻辑测试的 AdminService，
// 不触碰 DB（退避状态全在内存），并注入可控时钟。
func newLockoutTestService(now *time.Time) *AdminService {
	s := &AdminService{}
	s.nowFunc = func() time.Time { return *now }
	return s
}

func TestCheckLockout_NoFailures(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	if err := s.CheckLockout(); err != nil {
		t.Fatalf("CheckLockout with no failures: got %v, want nil", err)
	}
}

func TestRecordLoginFailure_LocksWithExponentialBackoff(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	// 第 1 次失败 → 锁定 2^1 = 2 秒。
	s.RecordLoginFailure()
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("after 1 failure: got %v, want ErrRateLimited", err)
	}

	// 推进 1 秒（仍在 2 秒锁定内）→ 仍锁定。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("at +1s of 2s lock: got %v, want ErrRateLimited", err)
	}

	// 推进到第 2 秒末（>= 锁定截止）→ 放行。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("at +2s of 2s lock: got %v, want nil", err)
	}
}

func TestRecordLoginFailure_BackoffGrows(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	// 累计 3 次失败 → 锁定 2^3 = 8 秒。
	s.RecordLoginFailure()
	s.RecordLoginFailure()
	s.RecordLoginFailure()

	// +7 秒仍锁定。
	now = now.Add(7 * time.Second)
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("at +7s of 8s lock: got %v, want ErrRateLimited", err)
	}

	// +8 秒放行。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("at +8s of 8s lock: got %v, want nil", err)
	}
}

func TestRecordLoginFailure_CapsAtOneHour(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	// 失败 20 次：2^20 秒远超 1 小时，必须封顶到 3600 秒。
	for i := 0; i < 20; i++ {
		s.RecordLoginFailure()
	}

	// +3599 秒仍锁定。
	now = now.Add(3599 * time.Second)
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("at +3599s of capped lock: got %v, want ErrRateLimited", err)
	}

	// +3600 秒放行（证明封顶为 1 小时，未无限增长）。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("at +3600s of capped lock: got %v, want nil", err)
	}
}

func TestResetLoginFailures_Unlocks(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	s.RecordLoginFailure()
	s.RecordLoginFailure()
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("after 2 failures: got %v, want ErrRateLimited", err)
	}

	// 重置后立即放行。
	s.ResetLoginFailures()
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("after reset: got %v, want nil", err)
	}

	// 重置还必须把计数归零：重置后第 1 次失败应只锁 2^1 = 2 秒，而非延续之前的指数。
	s.RecordLoginFailure()
	now = now.Add(2 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("first failure after reset should lock only 2s; at +2s got %v, want nil", err)
	}
}
