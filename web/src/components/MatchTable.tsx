import type { MatchView } from "@/api/types";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface MatchTableProps {
  matches: MatchView[];
  onDelete?: (id: number) => void;
  onRestore?: (id: number) => void;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function MatchTable({
  matches,
  onDelete,
  onRestore,
}: MatchTableProps) {
  if (matches.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">
        暂无对局记录
      </div>
    );
  }
  return (
    <Table data-testid="match-table">
      <TableHeader>
        <TableRow>
          <TableHead>对手</TableHead>
          <TableHead>结果</TableHead>
          <TableHead className="text-right">赛前</TableHead>
          <TableHead className="text-right">赛后</TableHead>
          <TableHead className="text-right">变化</TableHead>
          <TableHead>时间</TableHead>
          {(onDelete || onRestore) && <TableHead className="text-right">操作</TableHead>}
        </TableRow>
      </TableHeader>
      <TableBody>
        {matches.map((m) => (
          <TableRow key={m.id}>
            <TableCell className="font-medium">{m.opponent}</TableCell>
            <TableCell>
              <span
                className={cn(
                  "font-semibold",
                  m.result === "win" ? "text-emerald-400" : "text-rose-400",
                )}
              >
                {m.result === "win" ? "胜" : "负"}
              </span>
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {m.rating_before}
            </TableCell>
            <TableCell className="text-right tabular-nums">
              {m.rating_after}
            </TableCell>
            <TableCell
              className={cn(
                "text-right tabular-nums font-semibold",
                m.delta >= 0 ? "text-emerald-400" : "text-rose-400",
              )}
            >
              {m.delta >= 0 ? `+${m.delta}` : m.delta}
            </TableCell>
            <TableCell className="text-muted-foreground">
              {formatTime(m.played_at)}
            </TableCell>
            {(onDelete || onRestore) && (
              <TableCell className="text-right">
                {onDelete && (
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => onDelete(m.id)}
                  >
                    删除
                  </Button>
                )}
                {onRestore && (
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onRestore(m.id)}
                  >
                    恢复
                  </Button>
                )}
              </TableCell>
            )}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
