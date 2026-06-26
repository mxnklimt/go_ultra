import { client } from "@/api/client";
import type {
  MatchView,
  RecordMatchRequest,
  RecordMatchResponse,
  LeaderboardRow,
  CompareResult,
} from "@/api/types";

export async function recordMatch(
  body: RecordMatchRequest,
): Promise<RecordMatchResponse> {
  const res = await client.post<RecordMatchResponse>("/matches", body);
  return res.data;
}

export async function listGlobalMatches(params?: {
  limit?: number;
  offset?: number;
}): Promise<MatchView[]> {
  const res = await client.get<MatchView[]>("/matches", { params });
  return res.data;
}

export async function getLeaderboard(
  minGames = 0,
): Promise<LeaderboardRow[]> {
  const res = await client.get<LeaderboardRow[]>("/leaderboard", {
    params: { min_games: minGames },
  });
  return res.data;
}

export async function getCompare(
  usernames: string[],
  params?: { from?: string; to?: string },
): Promise<CompareResult> {
  const res = await client.get<CompareResult>("/compare", {
    params: { usernames: usernames.join(","), ...params },
  });
  return res.data;
}
