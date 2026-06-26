package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/service"

	"github.com/gin-gonic/gin"
)

// playerSvc 是 player handler 依赖的能力集合。由 *service.PlayerService 实现。
type playerSvc interface {
	GetByUsername(ctx context.Context, username string) (domain.Player, error)
	GetStats(ctx context.Context, playerID int64) (domain.Stats, error)
	ListByRating(ctx context.Context) ([]domain.Player, error)
}

// playerMatchSvc 是 player handler 依赖的对局相关能力。由 *service.MatchService 实现。
type playerMatchSvc interface {
	ListByPlayer(ctx context.Context, playerID int64, limit, offset int) ([]service.MatchView, error)
	History(ctx context.Context, playerID int64, createdAt time.Time) ([]service.HistoryPoint, error)
}

type playerHandler struct {
	player playerSvc
	match  playerMatchSvc
	// statsCounter 用于列表页拿每个玩家的胜负数，复用 player service 的 GetStats。
}

type playerListItem struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	Rating      int     `json:"rating"`
	Dan         int     `json:"dan"`
	GamesPlayed int     `json:"games_played"`
	WinRate     float64 `json:"win_rate"`
}

type playerDetail struct {
	ID        int64       `json:"id"`
	Username  string      `json:"username"`
	Rating    int         `json:"rating"`
	Dan       int         `json:"dan"`
	CreatedAt string      `json:"created_at"`
	Stats     playerStats `json:"stats"`
}

type playerStats struct {
	Wins          int     `json:"wins"`
	Losses        int     `json:"losses"`
	WinRate       float64 `json:"win_rate"`
	CurrentStreak int     `json:"current_streak"`
	LongestStreak int     `json:"longest_streak"`
}

type historyPointDTO struct {
	PlayedAt string `json:"played_at"`
	Rating   int    `json:"rating"`
}

type matchViewDTO struct {
	ID           int64  `json:"id"`
	Opponent     string `json:"opponent"`
	Result       string `json:"result"`
	RatingBefore int    `json:"rating_before"`
	RatingAfter  int    `json:"rating_after"`
	Delta        int    `json:"delta"`
	PlayedAt     string `json:"played_at"`
}

// handleListPlayers 返回所有玩家（按 rating DESC）及其统计。
func (h *playerHandler) handleListPlayers(c *gin.Context) {
	players, err := h.player.ListByRating(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]playerListItem, 0, len(players))
	for _, p := range players {
		st, err := h.player.GetStats(c.Request.Context(), p.ID)
		if err != nil {
			respondError(c, err)
			return
		}
		out = append(out, playerListItem{
			ID:          p.ID,
			Username:    p.Username,
			Rating:      p.Rating,
			Dan:         domain.Dan(p.Rating),
			GamesPlayed: st.Wins + st.Losses,
			WinRate:     st.WinRate,
		})
	}
	c.JSON(http.StatusOK, out)
}

// handleGetPlayer 返回单个玩家及其完整统计。
func (h *playerHandler) handleGetPlayer(c *gin.Context) {
	username := c.Param("username")
	p, err := h.player.GetByUsername(c.Request.Context(), username)
	if err != nil {
		respondError(c, err)
		return
	}
	st, err := h.player.GetStats(c.Request.Context(), p.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, playerDetail{
		ID:        p.ID,
		Username:  p.Username,
		Rating:    p.Rating,
		Dan:       domain.Dan(p.Rating),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
		Stats: playerStats{
			Wins:          st.Wins,
			Losses:        st.Losses,
			WinRate:       st.WinRate,
			CurrentStreak: st.CurrentStreak,
			LongestStreak: st.LongestStreak,
		},
	})
}

// handlePlayerHistory 返回玩家历史曲线点（service 已 prepend 起点）。
func (h *playerHandler) handlePlayerHistory(c *gin.Context) {
	username := c.Param("username")
	p, err := h.player.GetByUsername(c.Request.Context(), username)
	if err != nil {
		respondError(c, err)
		return
	}
	pts, err := h.match.History(c.Request.Context(), p.ID, p.CreatedAt)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]historyPointDTO, 0, len(pts))
	for _, pt := range pts {
		out = append(out, historyPointDTO{
			PlayedAt: pt.PlayedAt.UTC().Format(time.RFC3339),
			Rating:   pt.Rating,
		})
	}
	c.JSON(http.StatusOK, out)
}

// handlePlayerMatches 返回玩家对局流（分页）。
func (h *playerHandler) handlePlayerMatches(c *gin.Context) {
	username := c.Param("username")
	p, err := h.player.GetByUsername(c.Request.Context(), username)
	if err != nil {
		respondError(c, err)
		return
	}
	limit, offset := parseLimitOffset(c)
	views, err := h.match.ListByPlayer(c.Request.Context(), p.ID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]matchViewDTO, 0, len(views))
	for _, v := range views {
		out = append(out, toMatchViewDTO(v))
	}
	c.JSON(http.StatusOK, out)
}

func toMatchViewDTO(v service.MatchView) matchViewDTO {
	return matchViewDTO{
		ID:           v.ID,
		Opponent:     v.Opponent,
		Result:       v.Result,
		RatingBefore: v.RatingBefore,
		RatingAfter:  v.RatingAfter,
		Delta:        v.Delta,
		PlayedAt:     v.PlayedAt.UTC().Format(time.RFC3339),
	}
}

// parseLimitOffset 解析 limit/offset 查询参数，提供默认值与上限。
func parseLimitOffset(c *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
			if limit > 500 {
				limit = 500
			}
		}
	}
	if s := c.Query("offset"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
