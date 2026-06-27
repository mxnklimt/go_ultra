@echo off
setlocal enabledelayedexpansion
set "ROOT=%~dp0"

echo [start] preparing logs directory ...
if not exist "%ROOT%logs" mkdir "%ROOT%logs"

echo [start] launching go_ultra.exe (api :8080) ...
start "go_ultra-api" /B "%ROOT%server\go_ultra.exe"

echo [start] waiting for backend health at http://localhost:8080/api/healthz ...
set "HEALTHY="
for /L %%i in (1,1,30) do (
    if not defined HEALTHY (
        for /f %%c in ('curl -s -o NUL -w "%%{http_code}" http://localhost:8080/api/healthz') do set "CODE=%%c"
        if "!CODE!"=="200" (
            set "HEALTHY=1"
            echo [start] backend healthy ^(attempt %%i^).
        ) else (
            echo [start] not ready yet ^(attempt %%i, http=!CODE!^), retrying ...
            timeout /t 1 /nobreak >NUL
        )
    )
)

if not defined HEALTHY (
    echo [start] ERROR: backend did not become healthy after 30 attempts. Check logs\server.log
    endlocal
    exit /b 1
)

echo [start] launching caddy (reverse proxy + tls) ...
start "caddy" /B caddy run --config "%ROOT%Caddyfile"

echo [start] launching cloudflared tunnel (go-ultra) ...
start "cloudflared" /B cloudflared tunnel run go-ultra

echo [start] all services started.
echo [start]   api        : http://localhost:8080
echo [start]   caddy local : https://localhost (tls internal)
echo [start]   public      : via Cloudflare Tunnel
endlocal
