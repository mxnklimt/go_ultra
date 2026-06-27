package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/service"

	"github.com/gin-gonic/gin"
)

// leaderboardSvc 是 leaderboard handler 依赖的能力。由 *service.LeaderboardService 实现。
type leaderboardSvc interface {
	List(ctx context.Context, minGames int) ([]service.LeaderboardRow, error)
	CompareData(ctx context.Context, usernames []string) (service.CompareResult, error)
}

type leaderboardHandler struct {
	svc leaderboardSvc
}

type leaderboardRowDTO struct {
	Rank        int     `json:"rank"`
	Username    string  `json:"username"`
	Rating      float64 `json:"rating"`
	Dan         int     `json:"dan"`
	GamesPlayed int     `json:"games_played"`
	WinRate     float64 `json:"win_rate"`
}

type comparePointDTO struct {
	PlayedAt string  `json:"played_at"`
	Rating   float64 `json:"rating"`
}

type compareSeriesDTO struct {
	Username string            `json:"username"`
	Color    string            `json:"color"`
	Points   []comparePointDTO `json:"points"`
}

type headToHeadDTO struct {
	A     string `json:"a"`
	B     string `json:"b"`
	AWins int    `json:"a_wins"`
	BWins int    `json:"b_wins"`
}

type compareResponse struct {
	Series     []compareSeriesDTO `json:"series"`
	HeadToHead []headToHeadDTO    `json:"head_to_head"`
}

// handleLeaderboard 返回排行榜。
func (h *leaderboardHandler) handleLeaderboard(c *gin.Context) {
	minGames := 0
	if s := c.Query("min_games"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			minGames = n
		}
	}
	rows, err := h.svc.List(c.Request.Context(), minGames)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]leaderboardRowDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, leaderboardRowDTO{
			Rank:        r.Rank,
			Username:    r.Username,
			Rating:      r.Rating,
			Dan:         r.Dan,
			GamesPlayed: r.GamesPlayed,
			WinRate:     r.WinRate,
		})
	}
	c.JSON(http.StatusOK, out)
}

// handleCompare 返回多人对比数据。usernames 上限 10。
func (h *leaderboardHandler) handleCompare(c *gin.Context) {
	raw := c.Query("usernames")
	names := splitUsernames(raw)
	if len(names) == 0 {
		respondError(c, domain.ErrInvalidParam)
		return
	}
	if len(names) > 10 {
		respondError(c, domain.ErrInvalidParam)
		return
	}
	res, err := h.svc.CompareData(c.Request.Context(), names)
	if err != nil {
		respondError(c, err)
		return
	}

	series := make([]compareSeriesDTO, 0, len(res.Series))
	for _, s := range res.Series {
		pts := make([]comparePointDTO, 0, len(s.Points))
		for _, p := range s.Points {
			pts = append(pts, comparePointDTO{
				PlayedAt: p.PlayedAt.UTC().Format(time.RFC3339),
				Rating:   p.Rating,
			})
		}
		series = append(series, compareSeriesDTO{
			Username: s.Username,
			Color:    s.Color,
			Points:   pts,
		})
	}
	h2h := make([]headToHeadDTO, 0, len(res.HeadToHead))
	for _, x := range res.HeadToHead {
		h2h = append(h2h, headToHeadDTO{A: x.A, B: x.B, AWins: x.AWins, BWins: x.BWins})
	}
	c.JSON(http.StatusOK, compareResponse{Series: series, HeadToHead: h2h})
}

// splitUsernames 把逗号分隔的字符串拆成去重后的非空用户名列表。
func splitUsernames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
