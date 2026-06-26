package handler

import (
	"context"
	"net/http"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/session"

	"github.com/gin-gonic/gin"
)

// authPlayerService 是 auth handler 依赖的玩家会话相关能力集合。
// 由 *service.PlayerService 实现。
type authPlayerService interface {
	LoginOrCreate(ctx context.Context, username string) (domain.Player, error)
	GetByUsername(ctx context.Context, username string) (domain.Player, error)
	GetByID(ctx context.Context, playerID int64) (domain.Player, error)
	CreatePlayerSession(ctx context.Context, playerID int64) (token string, expiresAt time.Time, err error)
	DeletePlayerSession(ctx context.Context, token string) error
}

// authAdminService 是 admin 鉴权 handler 依赖的能力集合。由 *service.AdminService 实现。
type authAdminService interface {
	VerifyPassword(ctx context.Context, pw string) (bool, error)
	CreateAdminSession(ctx context.Context) (token string, expiresAt time.Time, err error)
	CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error)
	DeleteAdminSession(ctx context.Context, token string) error
}

type loginRequest struct {
	Username string `json:"username"`
}

type adminLoginRequest struct {
	Password string `json:"password"`
}

// playerDTO 是 player 的 JSON 表示（snake_case）。
type playerDTO struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Rating    int    `json:"rating"`
	Dan       int    `json:"dan"`
	CreatedAt string `json:"created_at"`
}

func toPlayerDTO(p domain.Player) playerDTO {
	return playerDTO{
		ID:        p.ID,
		Username:  p.Username,
		Rating:    p.Rating,
		Dan:       domain.Dan(p.Rating),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// setPlayerCookie 写入玩家会话 cookie。
func setPlayerCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.PlayerCookieName, token, int(ttl.Seconds()), "/", "", true, true)
}

func clearPlayerCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.PlayerCookieName, "", -1, "/", "", true, true)
}

func setAdminCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.AdminCookieName, token, int(ttl.Seconds()), "/", "", true, true)
}

func clearAdminCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.AdminCookieName, "", -1, "/", "", true, true)
}

type authHandler struct {
	player authPlayerService
	admin  authAdminService
}

// handleLogin 隐式注册或登录玩家，并设置会话 cookie。
func (h *authHandler) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}

	// 判断是否为新建：登录前先查是否已存在。
	existed := true
	if _, err := h.player.GetByUsername(c.Request.Context(), req.Username); err != nil {
		if de, ok := err.(*domain.Error); ok && de.Code == domain.ErrPlayerNotFound.Code {
			existed = false
		}
		// 其它错误（含校验类）交由 LoginOrCreate 统一处理。
	}

	p, err := h.player.LoginOrCreate(c.Request.Context(), req.Username)
	if err != nil {
		respondError(c, err)
		return
	}

	token, expiresAt, err := h.player.CreatePlayerSession(c.Request.Context(), p.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	setPlayerCookie(c, token, time.Until(expiresAt))

	status := http.StatusOK
	if !existed {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"player": toPlayerDTO(p)})
}

// handleLogout 删除当前会话并清除 cookie。
func (h *authHandler) handleLogout(c *gin.Context) {
	if token, err := c.Cookie(session.PlayerCookieName); err == nil && token != "" {
		_ = h.player.DeletePlayerSession(c.Request.Context(), token)
	}
	clearPlayerCookie(c)
	c.Status(http.StatusNoContent)
}

// handleMe 返回当前登录玩家信息（依赖 PlayerAuth 注入的 playerID）。
func (h *authHandler) handleMe(c *gin.Context) {
	username, ok := currentUsername(c, h.player)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"player": username})
}

// currentUsername 取出当前玩家并返回其 DTO；失败时已写好响应，返回 ok=false。
func currentUsername(c *gin.Context, p authPlayerService) (playerDTO, bool) {
	v, exists := c.Get(ctxPlayerID)
	if !exists {
		respondError(c, domain.ErrNotAuthenticated)
		return playerDTO{}, false
	}
	pid, _ := v.(int64)
	pl, err := p.GetByID(c.Request.Context(), pid)
	if err != nil {
		respondError(c, err)
		return playerDTO{}, false
	}
	return toPlayerDTO(pl), true
}

// handleAdminLogin 校验密码并创建管理员会话。
func (h *authHandler) handleAdminLogin(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}
	ok, err := h.admin.VerifyPassword(c.Request.Context(), req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		respondError(c, domain.ErrInvalidParam)
		return
	}
	token, expiresAt, err := h.admin.CreateAdminSession(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	setAdminCookie(c, token, time.Until(expiresAt))
	c.JSON(http.StatusOK, gin.H{"expires_at": expiresAt.UTC().Format(time.RFC3339)})
}

// handleAdminLogout 删除管理员会话并清除 cookie。
func (h *authHandler) handleAdminLogout(c *gin.Context) {
	if token, err := c.Cookie(session.AdminCookieName); err == nil && token != "" {
		_ = h.admin.DeleteAdminSession(c.Request.Context(), token)
	}
	clearAdminCookie(c)
	c.Status(http.StatusNoContent)
}

// handleAdminStatus 返回当前管理员会话状态（无需 AdminAuth）。
func (h *authHandler) handleAdminStatus(c *gin.Context) {
	token, err := c.Cookie(session.AdminCookieName)
	if err != nil || token == "" {
		c.JSON(http.StatusOK, gin.H{"authed": false})
		return
	}
	ok, expiresAt, err := h.admin.CheckAdminSession(c.Request.Context(), token)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		c.JSON(http.StatusOK, gin.H{"authed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"authed":     true,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}
