import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  getPlayer,
  getPlayerHistory,
  getPlayerMatches,
} from "@/api/players";
import RatingChart from "@/components/RatingChart";
import RankBadge from "@/components/RankBadge";
import MatchTable from "@/components/MatchTable";
import SubmitMatchDialog from "@/components/SubmitMatchDialog";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface PlayerOverviewProps {
  username: string;
  isSelf: boolean;
}

export default function PlayerOverview({
  username,
  isSelf,
}: PlayerOverviewProps) {
  const navigate = useNavigate();

  const detailQuery = useQuery({
    queryKey: ["player", username],
    queryFn: () => getPlayer(username),
    staleTime: 30_000,
  });
  const historyQuery = useQuery({
    queryKey: ["player-history", username],
    queryFn: () => getPlayerHistory(username),
    staleTime: 30_000,
  });
  const matchesQuery = useQuery({
    queryKey: ["player-matches", username],
    queryFn: () => getPlayerMatches(username, { limit: 20, offset: 0 }),
    staleTime: 30_000,
  });

  const detail = detailQuery.data;

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_320px]">
      <section className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">{username}</h1>
            {detail && <RankBadge rating={detail.rating} />}
          </div>
          {!isSelf && (
            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                navigate(`/compare?p=${encodeURIComponent(username)}`)
              }
            >
              📊 对比
            </Button>
          )}
        </div>
        <Card>
          <CardContent className="pt-6">
            {historyQuery.data ? (
              <RatingChart points={historyQuery.data} />
            ) : (
              <div className="flex h-[420px] items-center justify-center text-muted-foreground">
                加载中…
              </div>
            )}
          </CardContent>
        </Card>
      </section>

      <aside className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">统计</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            {detail ? (
              <>
                <Row label="当前等级分" value={String(detail.rating)} />
                <Row label="胜" value={String(detail.stats.wins)} />
                <Row label="负" value={String(detail.stats.losses)} />
                <Row
                  label="胜率"
                  value={`${(detail.stats.win_rate * 100).toFixed(1)}%`}
                />
                <Row
                  label="当前连胜"
                  value={String(detail.stats.current_streak)}
                />
                <Row
                  label="最长连胜"
                  value={String(detail.stats.longest_streak)}
                />
              </>
            ) : (
              <div className="text-muted-foreground">加载中…</div>
            )}
          </CardContent>
        </Card>

        {isSelf && (
          <SubmitMatchDialog
            trigger={<Button className="w-full">录入对局</Button>}
          />
        )}

        <Card>
          <CardHeader>
            <CardTitle className="text-base">最近对局</CardTitle>
          </CardHeader>
          <CardContent>
            <MatchTable matches={matchesQuery.data ?? []} />
          </CardContent>
        </Card>
      </aside>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium tabular-nums">{value}</span>
    </div>
  );
}
