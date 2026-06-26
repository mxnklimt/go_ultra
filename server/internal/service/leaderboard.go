package service

import (
	"context"
	"database/sql"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

// comparePalette 是对比图的 5 色板（与契约/前端 ECharts 5 色板一致）。
var comparePalette = []string{"#4a9eff", "#7fd6a3", "#8b5cf6", "#e0c47d", "#f08080"}

// LeaderboardRow 是排行榜的一行。
type LeaderboardRow struct {
	Rank        int
	Username    string
	Rating      int
	Dan         int
	GamesPlayed int
	WinRate     float64
}

// CompareSeries 是对比图中某玩家的一条曲线。
type CompareSeries struct {
	Username string
	Color    string
	Points   []HistoryPoint
}

// HeadToHead 是两名玩家的交手统计。
type HeadToHead struct {
	A     string
	B     string
	AWins int
	BWins int
}

// CompareResult 是 /compare 的完整结果。
type CompareResult struct {
	Series     []CompareSeries
	HeadToHead []HeadToHead
}

// LeaderboardService 负责排行榜与多人对比。
type LeaderboardService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewLeaderboardService 构造 LeaderboardService。
func NewLeaderboardService(q *sqlc.Queries, db *sql.DB) *LeaderboardService {
	return &LeaderboardService{q: q, db: db}
}

// List 返回排行榜。按 rating 降序赋连续 rank；min_games 用 >= 过滤（games < minGames 的玩家不出现）。
func (s *LeaderboardService) List(ctx context.Context, minGames int) ([]LeaderboardRow, error) {
	players, err := s.q.ListPlayersByRating(ctx)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	rows := make([]LeaderboardRow, 0, len(players))
	rank := 0
	for _, p := range players {
		counts, cerr := s.q.CountPlayerWinsLosses(ctx, sqlc.CountPlayerWinsLossesParams{
			WinnerID:   p.ID,
			LoserID:    p.ID,
			WinnerID_2: p.ID,
			LoserID_2:  p.ID,
		})
		if cerr != nil {
			return nil, domain.ErrInternal.WithCause(cerr)
		}
		wins := int(counts.Wins)
		losses := int(counts.Losses)
		games := wins + losses
		if games < minGames {
			continue
		}
		var winRate float64
		if games > 0 {
			winRate = float64(wins) / float64(games)
		}
		rank++
		rows = append(rows, LeaderboardRow{
			Rank:        rank,
			Username:    p.Username,
			Rating:      int(p.Rating),
			Dan:         domain.Dan(int(p.Rating)),
			GamesPlayed: games,
			WinRate:     winRate,
		})
	}
	return rows, nil
}

// CompareData 为每个用户名组装一条历史曲线（配 5 色板循环），并生成所有 C(n,2) 对的交手统计。
func (s *LeaderboardService) CompareData(ctx context.Context, usernames []string) (CompareResult, error) {
	type entry struct {
		player domain.Player
		points []HistoryPoint
	}
	entries := make([]entry, 0, len(usernames))
	idByName := make(map[string]int64, len(usernames))

	matchSvc := NewMatchService(s.q, s.db)

	for i, name := range usernames {
		row, err := s.q.GetPlayerByUsername(ctx, name)
		if err != nil {
			if err == sql.ErrNoRows {
				return CompareResult{}, domain.ErrPlayerNotFound
			}
			return CompareResult{}, domain.ErrInternal.WithCause(err)
		}
		p, cerr := toDomainPlayer(row)
		if cerr != nil {
			return CompareResult{}, domain.ErrInternal.WithCause(cerr)
		}
		points, herr := matchSvc.History(ctx, p.ID, p.CreatedAt)
		if herr != nil {
			return CompareResult{}, herr
		}
		entries = append(entries, entry{player: p, points: points})
		idByName[p.Username] = p.ID
		_ = i
	}

	series := make([]CompareSeries, 0, len(entries))
	for i, e := range entries {
		series = append(series, CompareSeries{
			Username: e.player.Username,
			Color:    comparePalette[i%len(comparePalette)],
			Points:   e.points,
		})
	}

	// 生成 C(n,2) 对，AWins/BWins 仅计未删除局。
	heads := make([]HeadToHead, 0)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			aName := entries[i].player.Username
			bName := entries[j].player.Username
			aID := entries[i].player.ID
			bID := entries[j].player.ID
			aWins, bWins, err := s.headToHead(ctx, aID, bID)
			if err != nil {
				return CompareResult{}, err
			}
			heads = append(heads, HeadToHead{A: aName, B: bName, AWins: aWins, BWins: bWins})
		}
	}

	return CompareResult{Series: series, HeadToHead: heads}, nil
}

// headToHead 统计 a 与 b 之间未删除对局里 a 的胜场与 b 的胜场。
func (s *LeaderboardService) headToHead(ctx context.Context, aID, bID int64) (aWins, bWins int, err error) {
	// 复用 ListPlayerMatches 取 a 的全部未删除对局，再筛对手为 b 的。
	rows, qerr := s.q.ListPlayerMatches(ctx, sqlc.ListPlayerMatchesParams{
		WinnerID: aID,
		LoserID:  aID,
		Limit:    1000000,
		Offset:   0,
	})
	if qerr != nil {
		return 0, 0, domain.ErrInternal.WithCause(qerr)
	}
	for _, m := range rows {
		// 仅统计 a 与 b 之间的对局。
		isPair := (m.WinnerID == aID && m.LoserID == bID) || (m.WinnerID == bID && m.LoserID == aID)
		if !isPair {
			continue
		}
		if m.WinnerID == aID {
			aWins++
		} else {
			bWins++
		}
	}
	return aWins, bWins, nil
}
