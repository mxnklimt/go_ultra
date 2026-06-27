import type { ReactNode } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { ChevronDown } from "lucide-react";
import { useAuth } from "@/hooks/useAuth";
import { logout } from "@/api/players";
import SubmitMatchDialog from "@/components/SubmitMatchDialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/me", label: "我的" },
  { to: "/leaderboard", label: "排行榜" },
  { to: "/compare", label: "对比" },
];

export default function Layout({ children }: { children: ReactNode }) {
  const { player } = useAuth();
  const navigate = useNavigate();
  const qc = useQueryClient();

  async function handleLogout() {
    await logout();
    qc.clear();
    navigate("/login", { replace: true });
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="container flex h-14 items-center justify-between">
          <div className="flex items-center gap-6">
            <NavLink to="/me" className="text-lg font-bold">
              🎯 go_ultra
            </NavLink>
            <nav className="flex items-center gap-1">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    cn(
                      "rounded-md px-3 py-1.5 text-sm transition-colors",
                      isActive
                        ? "bg-secondary text-secondary-foreground"
                        : "text-muted-foreground hover:text-foreground",
                    )
                  }
                >
                  {item.label}
                </NavLink>
              ))}
              <SubmitMatchDialog
                trigger={
                  <Button variant="ghost" size="sm" className="text-sm">
                    录入对局
                  </Button>
                }
              />
            </nav>
          </div>

          {player && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="gap-1">
                  {player.username}
                  <ChevronDown className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => navigate("/me")}>
                  个人主页
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => navigate("/admin")}>
                  管理面板
                </DropdownMenuItem>
                <DropdownMenuItem onClick={handleLogout}>登出</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </header>
      <main className="container py-6">{children}</main>
    </div>
  );
}
