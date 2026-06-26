import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import Layout from "@/components/Layout";
import RankBadge from "@/components/RankBadge";
import { getLeaderboard } from "@/api/matches";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import type { LeaderboardRow } from "@/api/types";

const PODIUM_STYLE = [
  "border-yellow-500/60",
  "border-zinc-400/60",
  "border-amber-700/60",
];

export default function Leaderboard() {
  const navigate = useNavigate();
  const { data: rows = [] } = useQuery({
    queryKey: ["leaderboard"],
    queryFn: () => getLeaderboard(0),
    staleTime: 30_000,
  });

  const top3 = rows.slice(0, 3);
  const rest = rows.slice(3);

  return (
    <Layout>
      <h1 className="mb-6 text-2xl font-bold">排行榜</h1>

      {top3.length > 0 && (
        <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
          {top3.map((row, i) => (
            <PodiumCard
              key={row.username}
              row={row}
              rankClass={PODIUM_STYLE[i]}
              onClick={() =>
                navigate(`/players/${encodeURIComponent(row.username)}`)
              }
            />
          ))}
        </div>
      )}

      {rest.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-16">名次</TableHead>
              <TableHead>玩家</TableHead>
              <TableHead>段位</TableHead>
              <TableHead className="text-right">等级分</TableHead>
              <TableHead className="text-right">局数</TableHead>
              <TableHead className="text-right">胜率</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rest.map((row) => (
              <TableRow
                key={row.username}
                className="cursor-pointer"
                onClick={() =>
                  navigate(`/players/${encodeURIComponent(row.username)}`)
                }
              >
                <TableCell className="tabular-nums">{row.rank}</TableCell>
                <TableCell className="font-medium">{row.username}</TableCell>
                <TableCell>
                  <RankBadge rating={row.rating} />
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {row.rating}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {row.games_played}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {(row.win_rate * 100).toFixed(0)}%
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {rows.length === 0 && (
        <div className="py-12 text-center text-muted-foreground">
          暂无排名数据
        </div>
      )}
    </Layout>
  );
}

function PodiumCard({
  row,
  rankClass,
  onClick,
}: {
  row: LeaderboardRow;
  rankClass: string;
  onClick: () => void;
}) {
  return (
    <Card
      className={cn("cursor-pointer border-2", rankClass)}
      onClick={onClick}
    >
      <CardContent className="flex flex-col items-center gap-2 py-6">
        <div className="text-3xl font-bold tabular-nums">#{row.rank}</div>
        <div className="text-lg font-semibold">{row.username}</div>
        <RankBadge rating={row.rating} />
        <div className="text-2xl font-bold tabular-nums">{row.rating}</div>
        <div className="text-xs text-muted-foreground">
          {row.games_played} 局 · 胜率 {(row.win_rate * 100).toFixed(0)}%
        </div>
      </CardContent>
    </Card>
  );
}
