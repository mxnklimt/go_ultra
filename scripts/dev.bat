@echo off
setlocal
set "ROOT=%~dp0.."

echo [dev] starting backend (go run) on :8080 ...
start "go_ultra-dev-api" cmd /k "cd /d "%ROOT%\server" && go run ./cmd/go_ultra"

echo [dev] starting frontend (vite dev) on :5173 ...
start "go_ultra-dev-web" cmd /k "cd /d "%ROOT%\web" && pnpm dev"

echo [dev] both started in separate windows.
echo [dev]   backend : http://localhost:8080/api/healthz
echo [dev]   frontend: http://localhost:5173
echo [dev] close each window (or Ctrl+C inside it) to stop.
endlocal
