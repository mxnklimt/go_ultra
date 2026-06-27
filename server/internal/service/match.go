package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

// RecordResult 是录入一局后返回给调用方的结果（相对提交者视角）。
type RecordResult struct {
	MatchID           int64
	WinnerDelta       float64
	LoserDelta        float64
	NewSelfRating     float64
	NewOpponentRating float64
}

// MatchView 是某个玩家视角下的一条对局展示数据。
type MatchView struct {
	ID           int64
	Opponent     string
	Result       string // 相对查询玩家："win" | "loss"
	RatingBefore float64
	RatingAfter  float64
	Delta        float64
	PlayedAt     time.Time
}

// HistoryPoint 是历史曲线上的一个 (时间, 分数) 点。
type HistoryPoint struct {
	PlayedAt time.Time
	Rating   float64
}

// MatchService 负责对局录入与查询。
type MatchService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewMatchService 构造 MatchService。
func NewMatchService(q *sqlc.Queries, db *sql.DB) *MatchService {
	return &MatchService{q: q, db: db}
}

// Record 录入一局对局。result="win" 表示提交者获胜（winner=submitter）。
// 整个读-算-写过程在一个事务内完成。
func (s *MatchService) Record(ctx context.Context, submitterID int64, opponentUsername string, result string, playedAt time.Time) (RecordResult, error) {
	// 开一个事务。db.New 的 DSN 设了 _txlock=immediate，故每个事务在 BEGIN 时即取写锁，
	// 配合 busy_timeout=5000 让并发录入串行化（而非死锁在锁升级上）；DB 层三条 CHECK 守护零和不变量。
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}
	defer func() { _ = tx.Rollback() }() // 提交成功后 Rollback 是 no-op

	qtx := s.q.WithTx(tx)

	// 查对手。
	opponent, err := qtx.GetPlayerByUsername(ctx, opponentUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecordResult{}, domain.ErrPlayerNotFound
		}
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}
	if opponent.ID == submitterID {
		return RecordResult{}, domain.ErrSelfMatch
	}

	// 查提交者。
	submitter, err := qtx.GetPlayerByID(ctx, submitterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecordResult{}, domain.ErrPlayerNotFound
		}
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	// 根据 result 决定谁是 winner。
	var winnerID, loserID int64
	var winnerBefore, loserBefore float64
	switch result {
	case "win":
		winnerID, loserID = submitter.ID, opponent.ID
		winnerBefore, loserBefore = submitter.Rating, opponent.Rating
	case "loss":
		winnerID, loserID = opponent.ID, submitter.ID
		winnerBefore, loserBefore = opponent.Rating, submitter.Rating
	default:
		return RecordResult{}, domain.ErrInvalidParam
	}

	delta := domain.ComputeDelta(winnerBefore, loserBefore)
	winnerAfter := winnerBefore + delta
	loserAfter := loserBefore - delta

	now := time.Now().UTC()
	created, err := qtx.CreateMatch(ctx, sqlc.CreateMatchParams{
		WinnerID:           winnerID,
		LoserID:            loserID,
		SubmitterID:        submitterID,
		WinnerRatingBefore: winnerBefore,
		LoserRatingBefore:  loserBefore,
		WinnerRatingAfter:  winnerAfter,
		LoserRatingAfter:   loserAfter,
		WinnerDelta:        delta,
		LoserDelta:         -delta,
		PlayedAt:           formatTime(playedAt),
		CreatedAt:          formatTime(now),
	})
	if err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	if err := qtx.UpdatePlayerRating(ctx, sqlc.UpdatePlayerRatingParams{
		ID:     winnerID,
		Rating: winnerAfter,
	}); err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}
	if err := qtx.UpdatePlayerRating(ctx, sqlc.UpdatePlayerRatingParams{
		ID:     loserID,
		Rating: loserAfter,
	}); err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	if err := tx.Commit(); err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	// 组装相对提交者视角的返回值。
	res := RecordResult{MatchID: created.ID}
	if result == "win" {
		res.WinnerDelta = delta
		res.LoserDelta = -delta
		res.NewSelfRating = winnerAfter
		res.NewOpponentRating = loserAfter
	} else {
		res.WinnerDelta = delta
		res.LoserDelta = -delta
		res.NewSelfRating = loserAfter // 提交者是 loser
		res.NewOpponentRating = winnerAfter
	}
	return res, nil
}

// ListGlobal 返回全局对局流（不含已删除），按 played_at DESC。
// Opponent/Result 等视角字段相对 winner 渲染（全局流无"当前玩家"概念，统一以 winner 为主体）。
func (s *MatchService) ListGlobal(ctx context.Context, limit, offset int) ([]MatchView, error) {
	rows, err := s.q.ListGlobalMatches(ctx, sqlc.ListGlobalMatchesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	views := make([]MatchView, 0, len(rows))
	for _, m := range rows {
		// 全局流以 winner 为主体：Result 恒为 "win"，Opponent 为 loser。
		opp, err := s.usernameOf(ctx, m.LoserID)
		if err != nil {
			return nil, err
		}
		playedAt, perr := parseTime(m.PlayedAt)
		if perr != nil {
			return nil, domain.ErrInternal.WithCause(perr)
		}
		views = append(views, MatchView{
			ID:           m.ID,
			Opponent:     opp,
			Result:       "win",
			RatingBefore: m.WinnerRatingBefore,
			RatingAfter:  m.WinnerRatingAfter,
			Delta:        m.WinnerDelta,
			PlayedAt:     playedAt,
		})
	}
	return views, nil
}

// ListByPlayer 返回指定玩家的对局，所有字段相对该玩家渲染。
func (s *MatchService) ListByPlayer(ctx context.Context, playerID int64, limit, offset int) ([]MatchView, error) {
	rows, err := s.q.ListPlayerMatches(ctx, sqlc.ListPlayerMatchesParams{
		WinnerID: playerID,
		LoserID:  playerID,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	views := make([]MatchView, 0, len(rows))
	for _, m := range rows {
		playedAt, perr := parseTime(m.PlayedAt)
		if perr != nil {
			return nil, domain.ErrInternal.WithCause(perr)
		}
		mv := MatchView{ID: m.ID, PlayedAt: playedAt}
		if m.WinnerID == playerID {
			oppName, oerr := s.usernameOf(ctx, m.LoserID)
			if oerr != nil {
				return nil, oerr
			}
			mv.Opponent = oppName
			mv.Result = "win"
			mv.RatingBefore = m.WinnerRatingBefore
			mv.RatingAfter = m.WinnerRatingAfter
			mv.Delta = m.WinnerDelta
		} else {
			oppName, oerr := s.usernameOf(ctx, m.WinnerID)
			if oerr != nil {
				return nil, oerr
			}
			mv.Opponent = oppName
			mv.Result = "loss"
			mv.RatingBefore = m.LoserRatingBefore
			mv.RatingAfter = m.LoserRatingAfter
			mv.Delta = m.LoserDelta
		}
		views = append(views, mv)
	}
	return views, nil
}

// History 返回该玩家的历史曲线点，开头 prepend (createdAt, DefaultRating) 作为起点。
func (s *MatchService) History(ctx context.Context, playerID int64, createdAt time.Time) ([]HistoryPoint, error) {
	rows, err := s.q.GetPlayerHistory(ctx, sqlc.GetPlayerHistoryParams{
		WinnerID: playerID,
		LoserID:  playerID,
	})
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	points := make([]HistoryPoint, 0, len(rows)+1)
	points = append(points, HistoryPoint{PlayedAt: createdAt.UTC(), Rating: domain.DefaultRating})
	for _, r := range rows {
		playedAt, perr := parseTime(r.PlayedAt)
		if perr != nil {
			return nil, domain.ErrInternal.WithCause(perr)
		}
		var rating float64
		if r.WinnerID == playerID {
			rating = r.WinnerRatingAfter
		} else {
			rating = r.LoserRatingAfter
		}
		points = append(points, HistoryPoint{PlayedAt: playedAt, Rating: rating})
	}
	return points, nil
}

// usernameOf 查某玩家的用户名（用于组装对局视角的对手名）。
// 注意：列表路径每行调用一次，是有意的 N+1；朋友圈规模（<100 人、页大小≤50）可接受，若未来扩容应改为 JOIN。
func (s *MatchService) usernameOf(ctx context.Context, id int64) (string, error) {
	p, err := s.q.GetPlayerByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrPlayerNotFound
		}
		return "", domain.ErrInternal.WithCause(err)
	}
	return p.Username, nil
}
