package middleware

import (
	"context"
	"net/http"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/session"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// PlayerSessionChecker 校验玩家会话 token，返回对应 playerID。
// ok=false 表示 token 无效或已过期（非系统错误）。
type PlayerSessionChecker interface {
	GetSession(ctx context.Context, token string) (playerID int64, ok bool, err error)
}

// AdminSessionChecker 校验管理员会话 token。
type AdminSessionChecker interface {
	CheckAdminSession(ctx context.Context, token string) (ok bool, expiresAt time.Time, err error)
}

// abort 把 domain.Error 写成统一 JSON 并终止链；非 domain.Error 一律 500。5xx 记录 Cause 便于排查。
func abort(c *gin.Context, err error) {
	de, ok := err.(*domain.Error)
	if !ok || de == nil {
		de = domain.ErrInternal.WithCause(err)
	}
	if de.Status >= http.StatusInternalServerError {
		if v, ok := c.Get(CtxLogger); ok {
			if lg, ok := v.(zerolog.Logger); ok {
				ev := lg.Error().Str("code", de.Code).Str("message", de.Message)
				if de.Cause != nil {
					ev = ev.Err(de.Cause)
				}
				ev.Msg("auth error")
			}
		}
	}
	c.AbortWithStatusJSON(de.Status, gin.H{
		"error": gin.H{"code": de.Code, "message": de.Message},
	})
}

// PlayerAuth 要求请求带有效的玩家会话 cookie，并把 playerID 注入 context。
func PlayerAuth(checker PlayerSessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(session.PlayerCookieName)
		if err != nil || token == "" {
			abort(c, domain.ErrNotAuthenticated)
			return
		}
		playerID, ok, err := checker.GetSession(c.Request.Context(), token)
		if err != nil {
			abort(c, domain.ErrInternal.WithCause(err))
			return
		}
		if !ok {
			abort(c, domain.ErrNotAuthenticated)
			return
		}
		c.Set(CtxPlayerID, playerID)
		c.Next()
	}
}

// AdminAuth 要求请求带有效的管理员会话 cookie。
func AdminAuth(checker AdminSessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(session.AdminCookieName)
		if err != nil || token == "" {
			abort(c, domain.ErrAdminRequired)
			return
		}
		ok, _, err := checker.CheckAdminSession(c.Request.Context(), token)
		if err != nil {
			abort(c, domain.ErrInternal.WithCause(err))
			return
		}
		if !ok {
			abort(c, domain.ErrAdminRequired)
			return
		}
		c.Next()
	}
}
