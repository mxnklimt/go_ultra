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

// matchSvc 是 match handler 依赖的录入/查询能力。由 *service.MatchService 实现。
type matchSvc interface {
	Record(ctx context.Context, submitterID int64, opponentUsername string, result string, playedAt time.Time) (service.RecordResult, error)
	ListGlobal(ctx context.Context, limit, offset int) ([]service.MatchView, error)
}

// adminMatchSvc 是删除/恢复/已删除列表能力。由 *service.AdminService 实现。
type adminMatchSvc interface {
	SoftDelete(ctx context.Context, matchID int64) error
	Restore(ctx context.Context, matchID int64) error
	ListDeleted(ctx context.Context) ([]domain.Match, error)
}

type matchHandler struct {
	match matchSvc
	admin adminMatchSvc
}

type recordMatchRequest struct {
	OpponentUsername string  `json:"opponent_username"`
	Result           string  `json:"result"`
	PlayedAt         *string `json:"played_at"`
}

type recordResultDTO struct {
	ID                int64 `json:"id"`
	WinnerDelta       int   `json:"winner_delta"`
	LoserDelta        int   `json:"loser_delta"`
	NewSelfRating     int   `json:"new_self_rating"`
	NewOpponentRating int   `json:"new_opponent_rating"`
}

type deletedMatchDTO struct {
	ID        int64  `json:"id"`
	WinnerID  int64  `json:"winner_id"`
	LoserID   int64  `json:"loser_id"`
	PlayedAt  string `json:"played_at"`
	DeletedAt string `json:"deleted_at"`
}

// handleRecordMatch 录入一局对局。
func (h *matchHandler) handleRecordMatch(c *gin.Context) {
	v, exists := c.Get(ctxPlayerID)
	if !exists {
		respondError(c, domain.ErrNotAuthenticated)
		return
	}
	submitterID, _ := v.(int64)

	var req recordMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}
	if req.Result != "win" && req.Result != "loss" {
		respondError(c, domain.ErrInvalidBody)
		return
	}

	playedAt := time.Now().UTC()
	if req.PlayedAt != nil && *req.PlayedAt != "" {
		t, err := time.Parse(time.RFC3339, *req.PlayedAt)
		if err != nil {
			respondError(c, domain.ErrInvalidParam.WithCause(err))
			return
		}
		playedAt = t.UTC()
		if playedAt.After(time.Now().UTC()) {
			respondError(c, domain.ErrInvalidParam)
			return
		}
	}

	res, err := h.match.Record(c.Request.Context(), submitterID, req.OpponentUsername, req.Result, playedAt)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, recordResultDTO{
		ID:                res.MatchID,
		WinnerDelta:       res.WinnerDelta,
		LoserDelta:        res.LoserDelta,
		NewSelfRating:     res.NewSelfRating,
		NewOpponentRating: res.NewOpponentRating,
	})
}

// handleListGlobalMatches 返回全局对局流（不含已删除）。
func (h *matchHandler) handleListGlobalMatches(c *gin.Context) {
	limit, offset := parseLimitOffset(c)
	views, err := h.match.ListGlobal(c.Request.Context(), limit, offset)
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

// handleDeleteMatch 软删除一局（仅管理员）。
func (h *matchHandler) handleDeleteMatch(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.admin.SoftDelete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleListDeletedMatches 返回已删除对局列表（仅管理员）。
func (h *matchHandler) handleListDeletedMatches(c *gin.Context) {
	matches, err := h.admin.ListDeleted(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]deletedMatchDTO, 0, len(matches))
	for _, m := range matches {
		dto := deletedMatchDTO{
			ID:       m.ID,
			WinnerID: m.WinnerID,
			LoserID:  m.LoserID,
			PlayedAt: m.PlayedAt.UTC().Format(time.RFC3339),
		}
		if m.DeletedAt != nil {
			dto.DeletedAt = m.DeletedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, dto)
	}
	c.JSON(http.StatusOK, out)
}

// handleRestoreMatch 恢复一局软删除（仅管理员）。
func (h *matchHandler) handleRestoreMatch(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.admin.Restore(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// parseIDParam 解析路径参数 :id。
func parseIDParam(c *gin.Context) (int64, error) {
	s := c.Param("id")
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrInvalidParam
	}
	return id, nil
}
