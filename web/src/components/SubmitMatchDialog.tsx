import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import PlayerCombobox from "@/components/PlayerCombobox";
import { listPlayers, getMe } from "@/api/players";
import { recordMatch } from "@/api/matches";
import { previewMatch } from "@/lib/elo-preview";
import { ApiError } from "@/api/client";
import type { MatchResult } from "@/api/types";

const schema = z.object({
  opponent_username: z.string().min(1, "请选择对手"),
  result: z.enum(["win", "loss"]),
});
type FormValues = z.infer<typeof schema>;

interface SubmitMatchDialogProps {
  trigger?: React.ReactNode;
}

export default function SubmitMatchDialog({ trigger }: SubmitMatchDialogProps) {
  const [open, setOpen] = useState(false);
  const qc = useQueryClient();

  const { data: me } = useQuery({ queryKey: ["me"], queryFn: getMe });
  const { data: players = [] } = useQuery({
    queryKey: ["players"],
    queryFn: listPlayers,
    staleTime: 30_000,
  });

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { opponent_username: "", result: "win" },
  });

  const opponentUsername = watch("opponent_username");
  const result = watch("result");
  const opponent = players.find((p) => p.username === opponentUsername);

  const preview =
    me && opponent
      ? previewMatch(me.rating, opponent.rating, result)
      : null;

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      recordMatch({
        opponent_username: values.opponent_username,
        result: values.result,
      }),
    onSuccess: (res) => {
      toast.success(
        `已录入：你 ${res.winner_delta >= 0 ? "+" : ""}${res.winner_delta.toFixed(2)} → ${res.new_self_rating.toFixed(2)}`,
      );
      qc.invalidateQueries({ queryKey: ["leaderboard"] });
      qc.invalidateQueries({ queryKey: ["me"] });
      qc.invalidateQueries({ queryKey: ["players"] });
      reset();
      setOpen(false);
    },
    onError: (err) => {
      const code = err instanceof ApiError ? err.code : "UNKNOWN";
      const msg =
        code === "SELF_MATCH"
          ? "不能和自己对局"
          : code === "PLAYER_NOT_FOUND"
            ? "对手不存在"
            : "录入失败";
      toast.error(msg);
    },
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? <Button data-testid="open-submit">录入对局</Button>}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>录入对局</DialogTitle>
        </DialogHeader>
        <form
          onSubmit={handleSubmit((v) => mutation.mutate(v))}
          className="space-y-4"
        >
          <div className="space-y-2">
            <Label>对手</Label>
            <PlayerCombobox
              value={opponentUsername}
              onChange={(u) =>
                setValue("opponent_username", u, { shouldValidate: true })
              }
              exclude={me ? [me.username] : []}
            />
            {errors.opponent_username && (
              <p className="text-xs text-rose-400">
                {errors.opponent_username.message}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <Label>结果</Label>
            <Select
              value={result}
              onValueChange={(v) =>
                setValue("result", v as MatchResult, { shouldValidate: true })
              }
            >
              <SelectTrigger data-testid="result-select">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="win">我赢了</SelectItem>
                <SelectItem value="loss">我输了</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {preview && (
            <div
              data-testid="elo-preview"
              className="rounded-md border bg-muted/40 p-3 text-sm"
            >
              <div className="flex justify-between">
                <span>我：{preview.self_before} →</span>
                <span
                  className={
                    preview.self_delta >= 0
                      ? "text-emerald-400"
                      : "text-rose-400"
                  }
                >
                  {preview.self_after.toFixed(2)}（
                  {preview.self_delta >= 0 ? "+" : ""}
                  {preview.self_delta.toFixed(2)}）
                </span>
              </div>
              <div className="flex justify-between">
                <span>{opponent?.username}：{preview.opponent_before} →</span>
                <span
                  className={
                    preview.opponent_delta >= 0
                      ? "text-emerald-400"
                      : "text-rose-400"
                  }
                >
                  {preview.opponent_after.toFixed(2)}（
                  {preview.opponent_delta >= 0 ? "+" : ""}
                  {preview.opponent_delta.toFixed(2)}）
                </span>
              </div>
            </div>
          )}

          <input type="hidden" {...register("opponent_username")} />

          <DialogFooter>
            <Button
              type="submit"
              disabled={mutation.isPending || !opponent}
              data-testid="submit-match"
            >
              {mutation.isPending ? "提交中…" : "提交"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
