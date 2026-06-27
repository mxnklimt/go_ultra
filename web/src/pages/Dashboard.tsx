import Layout from "@/components/Layout";
import PlayerOverview from "@/components/PlayerOverview";
import { useAuth } from "@/hooks/useAuth";

export default function Dashboard() {
  const { player } = useAuth();
  if (!player) {
    return (
      <Layout>
        <div className="text-muted-foreground">加载中…</div>
      </Layout>
    );
  }
  return (
    <Layout>
      <PlayerOverview username={player.username} isSelf />
    </Layout>
  );
}
