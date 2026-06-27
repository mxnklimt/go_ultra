import { client } from "@/api/client";
import type {
  Player,
  PlayerListItem,
  PlayerDetail,
  HistoryPoint,
  MatchView,
} from "@/api/types";

export async function login(username: string): Promise<Player> {
  const res = await client.post<{ player: Player }>("/login", { username });
  return res.data.player;
}

export async function logout(): Promise<void> {
  await client.post("/logout");
}

export async function getMe(): Promise<Player> {
  const res = await client.get<{ player: Player }>("/me");
  return res.data.player;
}

export async function listPlayers(): Promise<PlayerListItem[]> {
  const res = await client.get<PlayerListItem[]>("/players");
  return res.data;
}

export async function getPlayer(username: string): Promise<PlayerDetail> {
  const res = await client.get<PlayerDetail>(
    `/players/${encodeURIComponent(username)}`,
  );
  return res.data;
}

export async function getPlayerHistory(
  username: string,
  params?: { from?: string; to?: string },
): Promise<HistoryPoint[]> {
  const res = await client.get<HistoryPoint[]>(
    `/players/${encodeURIComponent(username)}/history`,
    { params },
  );
  return res.data;
}

export async function getPlayerMatches(
  username: string,
  params?: { limit?: number; offset?: number },
): Promise<MatchView[]> {
  const res = await client.get<MatchView[]>(
    `/players/${encodeURIComponent(username)}/matches`,
    { params },
  );
  return res.data;
}
