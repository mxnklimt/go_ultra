import { client } from "@/api/client";
import type {
  AdminStatus,
  AdminLoginResponse,
  MatchView,
} from "@/api/types";

export async function adminLogin(
  password: string,
): Promise<AdminLoginResponse> {
  const res = await client.post<AdminLoginResponse>("/admin/login", {
    password,
  });
  return res.data;
}

export async function adminLogout(): Promise<void> {
  await client.post("/admin/logout");
}

export async function getAdminStatus(): Promise<AdminStatus> {
  const res = await client.get<AdminStatus>("/admin/status");
  return res.data;
}

export async function deleteMatch(id: number): Promise<void> {
  await client.delete(`/matches/${id}`);
}

export async function listDeletedMatches(): Promise<MatchView[]> {
  const res = await client.get<MatchView[]>("/admin/matches/deleted");
  return res.data;
}

export async function restoreMatch(id: number): Promise<void> {
  await client.post(`/admin/matches/${id}/restore`);
}
