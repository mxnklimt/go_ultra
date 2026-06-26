import { useParams } from "react-router-dom";
import Layout from "@/components/Layout";
import PlayerOverview from "@/components/PlayerOverview";
import { useAuth } from "@/hooks/useAuth";

export default function PlayerDetail() {
  const { username = "" } = useParams();
  const { player } = useAuth();
  const isSelf = !!player && player.username === username;
  return (
    <Layout>
      <PlayerOverview username={username} isSelf={isSelf} />
    </Layout>
  );
}
