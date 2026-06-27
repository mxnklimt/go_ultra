package session

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Cookie 名常量。
const (
	PlayerCookieName = "go_ultra_session"
	AdminCookieName  = "go_ultra_admin"
)

// 会话有效期常量。
const (
	PlayerSessionTTL = 30 * 24 * time.Hour // 30 天，滑动续期
	AdminSessionTTL  = 30 * time.Minute
)

// NewToken 生成一个 32 字节的密码学随机 token，
// 以无填充的 base64 URL 安全编码返回（长度固定 43）。
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
