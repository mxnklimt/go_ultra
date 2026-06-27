import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";
import Layout from "@/components/Layout";
import CompareChart from "@/components/CompareChart";
import PlayerCombobox from "@/components/PlayerCombobox";
import { getCompare } from "@/api/matches";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const MAX_PLAYERS = 10;

export default function Compare() {
  const [searchParams, setSearchParams] = useSearchParams();

  const usernames = useMemo(() => {
    const p = searchParams.get("p");
    return p ? p.split(",").filter(Boolean) : [];
  }, [searchParams]);

  function setUsernames(next: string[]) {
    if (next.length === 0) {
      setSearchParams({});
    } else {
      setSearchParams({ p: next.join(",") });
    }
  }

  function addPlayer(username: string) {
    if (
      username &&
      !usernames.includes(username) &&
      usernames.length < MAX_PLAYERS
    ) {
      setUsernames([...usernames, username]);
    }
  }

  function removePlayer(username: string) {
    setUsernames(usernames.filter((u) => u !== username));
  }

  const { data } = useQuery({
    queryKey: ["compare", usernames],
    queryFn: () => getCompare(usernames),
    enabled: usernames.length >= 1,
    staleTime: 30_000,
  });

  return (
    <Layout>
      <h1 className="mb-6 text-2xl font-bold">多人对比</h1>
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
        <aside className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">选择玩家</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <PlayerCombobox
                value=""
                onChange={addPlayer}
                exclude={usernames}
                placeholder="添加玩家…"
              />
              <div className="space-y-2">
                {usernames.map((u) => (
                  <div
                    key={u}
                    className="flex items-center justify-between rounded-md border px-3 py-1.5 text-sm"
                  >
                    <span>{u}</span>
                    <button
                      type="button"
                      aria-label={`移除 ${u}`}
                      onClick={() => removePlayer(u)}
                      className="text-muted-foreground hover:text-foreground"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                ))}
                {usernames.length === 0 && (
                  <p className="text-xs text-muted-foreground">
                    至少添加一名玩家
                  </p>
                )}
              </div>
            </CardContent>
          </Card>
        </aside>

        <section className="space-y-6">
          <Card>
            <CardContent className="pt-6">
              {data && data.series.length > 0 ? (
                <CompareChart series={data.series} />
              ) : (
                <div className="flex h-[480px] items-center justify-center text-muted-foreground">
                  选择玩家以查看曲线
                </div>
              )}
            </CardContent>
          </Card>

          {data && data.head_to_head.length > 0 && (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {data.head_to_head.map((h) => (
                <Card key={`${h.a}-${h.b}`}>
                  <CardContent className="py-4">
                    <div className="mb-2 text-sm font-medium">
                      {h.a} vs {h.b}
                    </div>
                    <div className="flex items-baseline justify-between">
                      <span className="text-2xl font-bold tabular-nums text-emerald-400">
                        {h.a_wins}
                      </span>
                      <span className="text-muted-foreground">:</span>
                      <span className="text-2xl font-bold tabular-nums text-rose-400">
                        {h.b_wins}
                      </span>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </section>
      </div>
    </Layout>
  );
}
