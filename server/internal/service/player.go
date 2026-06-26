package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
	"go_ultra/internal/session"
)

// PlayerService 负责玩家账号的隐式注册、查询与统计。
type PlayerService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewPlayerService 构造 PlayerService。
func NewPlayerService(q *sqlc.Queries, db *sql.DB) *PlayerService {
	return &PlayerService{q: q, db: db}
}

// validateUsername trim 后校验长度为 3–32 个字符（按 rune 计数）。
func validateUsername(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(name)
	if n < 3 || n > 32 {
		return "", domain.ErrInvalidParam
	}
	return name, nil
}

// LoginOrCreate 校验并 trim 用户名；已存在则返回该玩家，否则按 DefaultRating 创建。幂等。
func (s *PlayerService) LoginOrCreate(ctx context.Context, username string) (domain.Player, error) {
	name, err := validateUsername(username)
	if err != nil {
		return domain.Player{}, err
	}

	// 先查（username 列 COLLATE NOCASE，大小写不敏感）。
	existing, err := s.q.GetPlayerByUsername(ctx, name)
	if err == nil {
		return toDomainPlayer(existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}

	// 不存在 → 创建。
	created, err := s.q.CreatePlayer(ctx, sqlc.CreatePlayerParams{
		Username: name,
		Rating:   int64(domain.DefaultRating),
	})
	if err != nil {
		// 并发下可能撞唯一约束：再查一次，保证幂等。
		again, qerr := s.q.GetPlayerByUsername(ctx, name)
		if qerr == nil {
			return toDomainPlayer(again)
		}
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}
	return toDomainPlayer(created)
}

// GetByUsername 按用户名查玩家；不存在返回 ErrPlayerNotFound。
func (s *PlayerService) GetByUsername(ctx context.Context, username string) (domain.Player, error) {
	name := strings.TrimSpace(username)
	row, err := s.q.GetPlayerByUsername(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Player{}, domain.ErrPlayerNotFound
		}
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}
	return toDomainPlayer(row)
}

// GetStats 统计指定玩家的胜负、胜率与连胜。
func (s *PlayerService) GetStats(ctx context.Context, playerID int64) (domain.Stats, error) {
	counts, err := s.q.CountPlayerWinsLosses(ctx, sqlc.CountPlayerWinsLossesParams{
		WinnerID:   playerID,
		LoserID:    playerID,
		WinnerID_2: playerID,
		LoserID_2:  playerID,
	})
	if err != nil {
		return domain.Stats{}, domain.ErrInternal.WithCause(err)
	}
	wins := int(counts.Wins)
	losses := int(counts.Losses)

	var winRate float64
	total := wins + losses
	if total > 0 {
		winRate = float64(wins) / float64(total)
	}

	// 遍历该玩家全部未删除对局（played_at ASC）算 streak。
	// limit 取一个足够大的常量（朋友圈规模 < 100 人、每天数十局，远不会触顶）。
	history, err := s.q.ListPlayerMatches(ctx, sqlc.ListPlayerMatchesParams{
		WinnerID: playerID,
		LoserID:  playerID,
		Limit:    1000000,
		Offset:   0,
	})
	if err != nil {
		return domain.Stats{}, domain.ErrInternal.WithCause(err)
	}

	// ListPlayerMatches 按 played_at DESC 返回；streak 计算需要时间升序，故倒序遍历。
	current := 0
	longest := 0
	run := 0
	for i := len(history) - 1; i >= 0; i-- {
		won := history[i].WinnerID == playerID
		if won {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
		// current streak：从最近一局（升序遍历的最后一条，即原始 i==0）回看。
		// 升序遍历到末尾时 run 即为"最近连胜"。
	}
	// 升序遍历结束后 run 恰好等于"从最近往前的连胜数"（因为最后一段连续胜局未被败局打断）。
	current = run

	return domain.Stats{
		Wins:          wins,
		Losses:        losses,
		WinRate:       winRate,
		CurrentStreak: current,
		LongestStreak: longest,
	}, nil
}

// ListByRating 返回所有玩家，按 rating 降序。
func (s *PlayerService) ListByRating(ctx context.Context) ([]domain.Player, error) {
	rows, err := s.q.ListPlayersByRating(ctx)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	players := make([]domain.Player, 0, len(rows))
	for _, r := range rows {
		p, cerr := toDomainPlayer(r)
		if cerr != nil {
			return nil, domain.ErrInternal.WithCause(cerr)
		}
		players = append(players, p)
	}
	return players, nil
}

// GetByID 按 ID 查玩家；不存在返回 ErrPlayerNotFound。
func (s *PlayerService) GetByID(ctx context.Context, playerID int64) (domain.Player, error) {
	row, err := s.q.GetPlayerByID(ctx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Player{}, domain.ErrPlayerNotFound
		}
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}
	return toDomainPlayer(row)
}

// CreatePlayerSession 为玩家创建一个会话，返回 token 与过期时间（PlayerSessionTTL）。
func (s *PlayerService) CreatePlayerSession(ctx context.Context, playerID int64) (string, time.Time, error) {
	token, err := session.NewToken()
	if err != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(session.PlayerSessionTTL)
	if err := s.q.CreateSession(ctx, sqlc.CreateSessionParams{
		Token:     token,
		PlayerID:  playerID,
		CreatedAt: formatTime(now),
		ExpiresAt: formatTime(expiresAt),
	}); err != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(err)
	}
	return token, expiresAt, nil
}

// GetSession 校验玩家会话 token；过期或不存在返回 ok=false（非错误）。
func (s *PlayerService) GetSession(ctx context.Context, token string) (int64, bool, error) {
	row, err := s.q.GetSession(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, domain.ErrInternal.WithCause(err)
	}
	expiresAt, perr := parseTime(row.ExpiresAt)
	if perr != nil {
		return 0, false, domain.ErrInternal.WithCause(perr)
	}
	if !time.Now().UTC().Before(expiresAt) {
		// 已过期：惰性删除并视为未认证。
		_ = s.q.DeleteSession(ctx, token)
		return 0, false, nil
	}
	return row.PlayerID, true, nil
}

// DeletePlayerSession 删除指定会话 token（登出）。幂等。
func (s *PlayerService) DeletePlayerSession(ctx context.Context, token string) error {
	if err := s.q.DeleteSession(ctx, token); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}
