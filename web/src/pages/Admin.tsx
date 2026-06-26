import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import Layout from "@/components/Layout";
import { listDeletedMatches, restoreMatch } from "@/api/admin";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

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

export default function Admin() {
  const qc = useQueryClient();
  const { data: deleted = [] } = useQuery({
    queryKey: ["admin-deleted-matches"],
    queryFn: listDeletedMatches,
    staleTime: 10_000,
  });

  const restoreMutation = useMutation({
    mutationFn: (id: number) => restoreMatch(id),
    onSuccess: () => {
      toast.success("已恢复对局");
      qc.invalidateQueries({ queryKey: ["admin-deleted-matches"] });
      qc.invalidateQueries({ queryKey: ["leaderboard"] });
    },
    onError: () => toast.error("恢复失败"),
  });

  return (
    <Layout>
      <h1 className="mb-6 text-2xl font-bold">管理员面板</h1>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">已删除对局</CardTitle>
        </CardHeader>
        <CardContent>
          {deleted.length === 0 ? (
            <div className="py-8 text-center text-sm text-muted-foreground">
              暂无已删除对局
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>胜者 ID</TableHead>
                  <TableHead>败者 ID</TableHead>
                  <TableHead>对局时间</TableHead>
                  <TableHead>删除时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deleted.map((m) => (
                  <TableRow key={m.id}>
                    <TableCell className="tabular-nums">{m.id}</TableCell>
                    <TableCell className="tabular-nums">{m.winner_id}</TableCell>
                    <TableCell className="tabular-nums">{m.loser_id}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatTime(m.played_at)}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatTime(m.deleted_at)}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => restoreMutation.mutate(m.id)}
                        disabled={restoreMutation.isPending}
                      >
                        恢复
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </Layout>
  );
}
