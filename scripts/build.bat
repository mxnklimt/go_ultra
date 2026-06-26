@echo off
setlocal
set "ROOT=%~dp0.."

echo [build] building frontend (pnpm build) ...
pushd "%ROOT%\web"
call pnpm install --frozen-lockfile
if errorlevel 1 (
    echo [build] FAILED: pnpm install
    popd
    exit /b 1
)
call pnpm build
if errorlevel 1 (
    echo [build] FAILED: pnpm build
    popd
    exit /b 1
)
popd

echo [build] building backend (go build -> server/go_ultra.exe) ...
pushd "%ROOT%\server"
go build -o go_ultra.exe ./cmd/go_ultra
if errorlevel 1 (
    echo [build] FAILED: go build
    popd
    exit /b 1
)
popd

echo [build] done.
echo [build]   frontend dist: %ROOT%\web\dist
echo [build]   backend  exe : %ROOT%\server\go_ultra.exe
endlocal
