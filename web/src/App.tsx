import { Routes, Route, Navigate } from "react-router-dom";
import { useAuth } from "@/hooks/useAuth";
import AuthGuard from "@/components/AuthGuard";
import AdminGuard from "@/components/AdminGuard";
import Login from "@/pages/Login";
import Dashboard from "@/pages/Dashboard";
import Leaderboard from "@/pages/Leaderboard";
import PlayerDetail from "@/pages/PlayerDetail";
import Compare from "@/pages/Compare";
import Admin from "@/pages/Admin";

function RootRedirect() {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center text-muted-foreground">
        加载中…
      </div>
    );
  }
  return <Navigate to={isAuthenticated ? "/me" : "/login"} replace />;
}

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<RootRedirect />} />
      <Route path="/login" element={<Login />} />
      <Route
        path="/me"
        element={
          <AuthGuard>
            <Dashboard />
          </AuthGuard>
        }
      />
      <Route
        path="/leaderboard"
        element={
          <AuthGuard>
            <Leaderboard />
          </AuthGuard>
        }
      />
      <Route
        path="/players/:username"
        element={
          <AuthGuard>
            <PlayerDetail />
          </AuthGuard>
        }
      />
      <Route
        path="/compare"
        element={
          <AuthGuard>
            <Compare />
          </AuthGuard>
        }
      />
      <Route
        path="/admin"
        element={
          <AuthGuard>
            <AdminGuard>
              <Admin />
            </AdminGuard>
          </AuthGuard>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
