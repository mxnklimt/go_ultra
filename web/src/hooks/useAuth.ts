import { useQuery } from "@tanstack/react-query";
import { getMe } from "@/api/players";
import { ApiError } from "@/api/client";
import type { Player } from "@/api/types";

export function useAuth() {
  const query = useQuery<Player, ApiError>({
    queryKey: ["me"],
    queryFn: getMe,
    retry: false,
    staleTime: 60_000,
  });

  return {
    player: query.data ?? null,
    isLoading: query.isLoading,
    isAuthenticated: !!query.data,
    error: query.error,
  };
}
