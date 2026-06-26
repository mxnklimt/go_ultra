@echo off
setlocal

echo [stop] stopping go_ultra.exe ...
taskkill /IM go_ultra.exe /F /T 2>NUL
if errorlevel 1 (echo [stop]   go_ultra.exe not running.) else (echo [stop]   go_ultra.exe stopped.)

echo [stop] stopping caddy.exe ...
taskkill /IM caddy.exe /F /T 2>NUL
if errorlevel 1 (echo [stop]   caddy.exe not running.) else (echo [stop]   caddy.exe stopped.)

echo [stop] stopping cloudflared.exe ...
taskkill /IM cloudflared.exe /F /T 2>NUL
if errorlevel 1 (echo [stop]   cloudflared.exe not running.) else (echo [stop]   cloudflared.exe stopped.)

echo [stop] done.
endlocal
