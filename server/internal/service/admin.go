package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
	"go_ultra/internal/session"
)

const adminPasswordHashKey = "admin_password_hash"

// AdminService 负责管理员密码、会话与对局软删除/恢复。
type AdminService struct {
	q  *sqlc.Queries
	db *sql.DB

	// 登录失败指数退避状态。设计取舍：
	//  1. 全局锁定（非按 IP）——系统仅一个管理员，全局计数让攻击者无法靠轮换 IP 绕过退避。
	//  2. 状态存内存，进程重启清零——单机朋友圈部署可接受；重启需本机访问权限，不构成穷举通道。
	//  3. nowFunc 可注入以便测试，零值回退 time.Now，避免测试依赖真实时钟与 sleep。
	mu          sync.Mutex
	failCount   int
	lockedUntil time.Time
	nowFunc     func() time.Time
}

// NewAdminService 构造 AdminService。
func NewAdminService(q *sqlc.Queries, db *sql.DB) *AdminService {
	return &AdminService{q: q, db: db}
}

// EnsurePassword 确保存在管理员密码。
// 若 settings 无 admin_password_hash：用 GenerateAdminPassword 生成可读明文，bcrypt 后存入，返回 (明文, true, nil)。
// 已存在：返回 ("", false, nil)。
func (s *AdminService) EnsurePassword(ctx context.Context) (string, bool, error) {
	_, err := s.q.GetSetting(ctx, adminPasswordHashKey)
	if err == nil {
		return "", false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, domain.ErrInternal.WithCause(err)
	}

	plaintext, hash, gerr := GenerateAdminPassword()
	if gerr != nil {
		return "", false, domain.ErrInternal.WithCause(gerr)
	}
	if serr := s.q.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   adminPasswordHashKey,
		Value: hash,
	}); serr != nil {
		return "", false, domain.ErrInternal.WithCause(serr)
	}
	return plaintext, true, nil
}

// VerifyPassword 校验明文密码是否匹配存储的 bcrypt 哈希。
func (s *AdminService) VerifyPassword(ctx context.Context, pw string) (bool, error) {
	hash, err := s.q.GetSetting(ctx, adminPasswordHashKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, domain.ErrInternal.WithCause(err)
	}
	if cmpErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); cmpErr != nil {
		return false, nil
	}
	return true, nil
}

// CreateAdminSession 生成一个 30 分钟有效的管理员会话并落库。
func (s *AdminService) CreateAdminSession(ctx context.Context) (string, time.Time, error) {
	token, err := session.NewToken()
	if err != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(err)
	}
	now := time.Now().UTC()
	// expires_at 以 RFC3339（秒精度）落库；返回值同样截断到秒，
	// 以保证调用方持有的过期时间与后续 CheckAdminSession 从库中读回的值一致（Equal）。
	expiresAt := now.Add(session.AdminSessionTTL).Truncate(time.Second)
	if serr := s.q.CreateAdminSession(ctx, sqlc.CreateAdminSessionParams{
		Token:     token,
		CreatedAt: formatTime(now),
		ExpiresAt: formatTime(expiresAt),
	}); serr != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(serr)
	}
	return token, expiresAt, nil
}

// CheckAdminSession 校验 token 对应的会话是否存在且未过期。
func (s *AdminService) CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error) {
	row, err := s.q.GetAdminSession(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, domain.ErrInternal.WithCause(err)
	}
	expiresAt, perr := parseTime(row.ExpiresAt)
	if perr != nil {
		return false, time.Time{}, domain.ErrInternal.WithCause(perr)
	}
	if time.Now().UTC().After(expiresAt) {
		return false, expiresAt, nil
	}
	return true, expiresAt, nil
}

// SoftDelete 软删除一局对局，deleted_by 置 NULL（管理员非 player）。
func (s *AdminService) SoftDelete(ctx context.Context, matchID int64) error {
	if err := s.q.SoftDeleteMatch(ctx, sqlc.SoftDeleteMatchParams{
		DeletedAt: sql.NullString{String: formatTime(time.Now().UTC()), Valid: true},
		DeletedBy: sql.NullInt64{Valid: false},
		ID:        matchID,
	}); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}

// Restore 取消一局对局的软删除（幂等）。
func (s *AdminService) Restore(ctx context.Context, matchID int64) error {
	if err := s.q.RestoreMatch(ctx, matchID); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}

// ListDeleted 返回所有已软删除的对局，按 deleted_at DESC。
// （底层 sqlc 查询名为 ListDeletedMatches；service 方法名按 http 层接口约定为 ListDeleted。）
func (s *AdminService) ListDeleted(ctx context.Context) ([]domain.Match, error) {
	rows, err := s.q.ListDeletedMatches(ctx)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	matches := make([]domain.Match, 0, len(rows))
	for _, r := range rows {
		m, cerr := toDomainMatch(r)
		if cerr != nil {
			return nil, domain.ErrInternal.WithCause(cerr)
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// DeleteAdminSession 删除指定管理员会话 token（登出）。幂等。
func (s *AdminService) DeleteAdminSession(ctx context.Context, token string) error {
	if err := s.q.DeleteAdminSession(ctx, token); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}

// adminPasswordAlphabet 是可读、可输入的密码字符集（去除易混字符 0/O/1/l/I）。
const adminPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// GenerateAdminPassword 生成 16 位可读随机明文密码及其 bcrypt 哈希。
// 供 EnsurePassword 与 ResetPassword（阶段 7）共用，保证两条路径产出格式一致、字符可输入。
func GenerateAdminPassword() (plaintext, hash string, err error) {
	const n = 16
	buf := make([]byte, n)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = adminPasswordAlphabet[int(b)%len(adminPasswordAlphabet)]
	}
	plaintext = string(out)
	h, herr := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if herr != nil {
		return "", "", herr
	}
	return plaintext, string(h), nil
}

// adminLockoutCap 是单次锁定时长的封顶（spec §8.1：封顶 1 小时）。
const adminLockoutCap = time.Hour

// now 返回当前时间，优先用可注入的 nowFunc（测试用），否则回退 time.Now。
// 调用方必须已持有 s.mu。
func (s *AdminService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

// CheckLockout 在当前处于锁定窗口内时返回 domain.ErrRateLimited，否则返回 nil。
// 应在校验管理员密码之前调用。
func (s *AdminService) CheckLockout() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.now().Before(s.lockedUntil) {
		return domain.ErrRateLimited
	}
	return nil
}

// RecordLoginFailure 记录一次密码校验失败：失败次数自增，并把锁定截止时间
// 设为 now + min(2^failCount 秒, 1 小时)。
func (s *AdminService) RecordLoginFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCount++
	if s.failCount > 63 {
		s.failCount = 63
	}
	backoff := time.Duration(1<<uint(s.failCount)) * time.Second
	// backoff<=0 防溢出：failCount≥55 时 (1<<n)*1e9(=2^9*5^9) 低 64 位为 0，int64 乘积回绕为 0；
	// failCount≥63 时左移触及符号位致负值。两种情况都封顶到 adminLockoutCap(1h)。
	if backoff > adminLockoutCap || backoff <= 0 {
		backoff = adminLockoutCap
	}
	s.lockedUntil = s.now().Add(backoff)
}

// ResetPassword 强制重新生成管理员密码（覆盖已有 admin_password_hash），返回新明文。
func (s *AdminService) ResetPassword(ctx context.Context) (string, error) {
	plaintext, hash, err := GenerateAdminPassword()
	if err != nil {
		return "", domain.ErrInternal.WithCause(err)
	}
	if err := s.q.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   adminPasswordHashKey,
		Value: hash,
	}); err != nil {
		return "", domain.ErrInternal.WithCause(err)
	}
	return plaintext, nil
}

// ResetLoginFailures 在登录成功后清空失败计数与锁定窗口。
func (s *AdminService) ResetLoginFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCount = 0
	s.lockedUntil = time.Time{}
}
