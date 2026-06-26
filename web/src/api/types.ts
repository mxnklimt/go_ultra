// 所有字段命名与后端 JSON（snake_case）一一对应，前端不做转换。

export interface Player {
  id: number;
  username: string;
  rating: number;
  dan: number;
  created_at: string;
}

export interface PlayerStats {
  wins: number;
  losses: number;
  win_rate: number;
  current_streak: number;
  longest_streak: number;
}

export interface PlayerDetail {
  id: number;
  username: string;
  rating: number;
  dan: number;
  created_at: string;
  stats: PlayerStats;
}

export interface LeaderboardRow {
  rank: number;
  username: string;
  rating: number;
  dan: number;
  games_played: number;
  win_rate: number;
}

export interface HistoryPoint {
  played_at: string;
  rating: number;
}

export type MatchResult = "win" | "loss";

export interface MatchView {
  id: number;
  opponent: string;
  result: MatchResult;
  rating_before: number;
  rating_after: number;
  delta: number;
  played_at: string;
}

export interface RecordMatchRequest {
  opponent_username: string;
  result: MatchResult;
  played_at?: string;
}

export interface RecordMatchResponse {
  id: number;
  winner_delta: number;
  loser_delta: number;
  new_self_rating: number;
  new_opponent_rating: number;
}

export interface CompareSeries {
  username: string;
  color: string;
  points: HistoryPoint[];
}

export interface HeadToHead {
  a: string;
  b: string;
  a_wins: number;
  b_wins: number;
}

export interface CompareResult {
  series: CompareSeries[];
  head_to_head: HeadToHead[];
}

export interface AdminStatus {
  authed: boolean;
  expires_at?: string;
}

export interface AdminLoginResponse {
  expires_at: string;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}
