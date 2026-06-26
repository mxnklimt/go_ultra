import { useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { getAdminStatus, adminLogin } from "@/api/admin";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ApiError } from "@/api/client";

export default function AdminGuard({ children }: { children: ReactNode }) {
  const qc = useQueryClient();
  const [password, setPassword] = useState("");
  const { data, isLoading } = useQuery({
    queryKey: ["admin-status"],
    queryFn: getAdminStatus,
    retry: false,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await adminLogin(password);
      await qc.invalidateQueries({ queryKey: ["admin-status"] });
      toast.success("管理员已登录");
    } catch (err) {
      const code = err instanceof ApiError ? err.code : "UNKNOWN";
      toast.error(code === "RATE_LIMITED" ? "尝试过于频繁，请稍后" : "密码错误");
    }
  }

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center text-muted-foreground">
        加载中…
      </div>
    );
  }
  if (!data?.authed) {
    return (
      <div className="flex h-screen items-center justify-center">
        <form
          onSubmit={handleSubmit}
          className="w-80 space-y-4 rounded-lg border bg-card p-6"
        >
          <h2 className="text-lg font-semibold">管理员登录</h2>
          <div className="space-y-2">
            <Label htmlFor="admin-password">密码</Label>
            <Input
              id="admin-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="off"
            />
          </div>
          <Button type="submit" className="w-full">
            登录
          </Button>
        </form>
      </div>
    );
  }
  return <>{children}</>;
}
