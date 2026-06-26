# go_ultra

通用 1v1 竞技游戏等级分系统。

- 后端：Go + Gin + SQLite + sqlc
- 前端：React 18 + TypeScript + Vite + Tailwind + shadcn/ui + ECharts
- 部署：Windows 本机 + Caddy 反代 + Cloudflare Tunnel

设计文档：[docs/superpowers/specs/2026-06-25-go-ultra-design.md](docs/superpowers/specs/2026-06-25-go-ultra-design.md)

## 运维（Operations）

### 一、Cloudflare Tunnel 一次性配置

公网访问通过 Cloudflare Tunnel 暴露本机的 Caddy（见 §3.1 拓扑）。以下步骤**只需做一次**，之后由 `cloudflared` Windows 服务自启。

1. 安装 cloudflared（Windows）：从 `https://github.com/cloudflare/cloudflared/releases` 下载 `cloudflared-windows-amd64.exe`，重命名为 `cloudflared.exe` 放入 `PATH`。验证：
   ```bat
   cloudflared --version
   ```
   预期输出形如：
   ```
   cloudflared version 2024.x.x (built ...)
   ```

2. 在 Cloudflare **Zero Trust** 控制台创建 Tunnel 并拿到 token：
   - 登录 `https://one.dash.cloudflare.com/` → 左侧 **Networks → Tunnels**（旧路径为 Access → Tunnels）。
   - 点击 **Create a tunnel** → 类型选 **Cloudflared** → 隧道名填 `go-ultra` → **Save tunnel**。
   - 在 "Install and run a connector" 页面选择 **Windows**，页面会显示一条形如下面的命令，**其中的长字符串就是 token**：
     ```
     cloudflared.exe service install eyJhIjoi...（很长的 base64 token）
     ```
   - 在 **Public Hostname** 标签为该隧道配置一条记录：
     - Subdomain / Domain：填你要对外暴露的域名（如 `go-ultra.example.com`，域名需已托管在 Cloudflare）。
     - Service：`Type = HTTPS`，`URL = localhost:443`（指向本机 Caddy）。
     - 在该 Public Hostname 的 **Additional application settings → TLS** 中，把 **No TLS Verify 打开**（因为 Caddy 用的是 `tls internal` 自签证书）。
     - **Save hostname**。

3. 把 cloudflared 安装为 Windows 服务（开机自启）。复制上一步页面给出的完整命令并以**管理员身份**运行的命令行执行：
   ```bat
   cloudflared.exe service install eyJhIjoi...（粘贴你自己的 token）
   ```
   预期输出包含：
   ```
   Successfully installed cloudflared!
   ```
   此后服务名为 `Cloudflared`，开机自动连接隧道。手动管理：
   ```bat
   sc query cloudflared          rem 查看服务状态
   sc stop  cloudflared          rem 停止
   sc start cloudflared          rem 启动
   cloudflared service uninstall rem 卸载服务
   ```

> 说明：若用 `start.bat` 里的 `cloudflared tunnel run go-ultra` 前台方式运行隧道，则**不要**同时安装服务，二者择一，避免两个连接器抢占同一隧道。`start.bat`/`stop.bat` 面向前台方式；若你装了服务，关闭请用 `sc stop cloudflared`。

### 二、首次启动如何获取管理员密码

首次启动时（`settings` 表中尚无 `admin_password_hash`），后端会**随机生成一个 16 位管理员密码**，用 bcrypt 存入数据库，并把**明文**输出到两处（明文只在这一次生成时出现，之后无法再次取回）：

1. **进程 stdout**：在 `go_ultra.exe` 的控制台输出中查找形如：
   ```
   ============================================================
   ADMIN PASSWORD (generated, shown only once): X7k9Qm2Lp4Rt8Vw
   also written to logs/admin_password.txt
   ============================================================
   ```
2. **文件 `logs/admin_password.txt`**：内容就是该明文密码。
   ```bat
   type logs\admin_password.txt
   ```

**文件权限说明（重要）：** `logs/admin_password.txt` 含明文密码，请妥善保管：
- 读到密码并妥善保存后，建议立即删除该文件：
  ```bat
  del logs\admin_password.txt
  ```
- 若需保留，至少把权限收紧为仅当前用户可读（用 `icacls` 移除继承并只授当前用户）：
  ```bat
  icacls logs\admin_password.txt /inheritance:r /grant:r "%USERNAME%:R"
  ```
  验证权限：
  ```bat
  icacls logs\admin_password.txt
  ```
  预期仅列出当前用户（`%USERNAME%`）的 `(R)` 项，无 `Everyone` / `Users` 等。
- 该文件**不应提交到 git**（确保仓库根 `.gitignore` 含 `logs/`）。

忘记密码时见下文"重置管理员密码"。

### 三、手动备份

本系统 MVP **不自动备份**（见 §10 风险表）。数据全部在单个 SQLite 文件中，备份 = 复制该文件。

- 数据库文件默认路径（随启动配置而定，下面以默认为例）：
  ```
  E:\go_ultra\server\go_ultra.db
  ```
- 备份前**先停止服务**（避免复制到写入中途的文件）：
  ```bat
  stop.bat
  copy "E:\go_ultra\server\go_ultra.db" "E:\go_ultra\backups\go_ultra-%date:~0,4%%date:~5,2%%date:~8,2%.db"
  ```
  （`backups` 目录不存在请先 `mkdir E:\go_ultra\backups`。）
- 注意：WAL 模式（`journal_mode=WAL`）下还会有 `go_ultra.db-wal` 与 `go_ultra.db-shm` 两个伴随文件。**停止服务后**这些会被合并回主库，此时只需复制 `.db` 主文件即可；若在运行中复制，请把三个文件一起复制。
- 恢复：停止服务 → 用备份文件覆盖回 `server\go_ultra.db` → 重新 `start.bat`。

### 四、重置管理员密码

忘记或需要轮换管理员密码时，用内置子命令（实现见后端 `cmd/go_ultra/main.go` 的 `reset-admin-password`）。**先停止正在运行的后端**（避免数据库写冲突）：

```bat
stop.bat
cd server
go_ultra.exe reset-admin-password
```
预期输出形如（同样会写入 `logs/admin_password.txt`）：
```
============================================================
ADMIN PASSWORD (reset, shown only once): N3p8Zq1Wd6Yb4Hs
also written to logs/admin_password.txt
============================================================
```
随后重新 `start.bat`，用新密码登录管理面板。旧密码立即失效。

该子命令不接受额外参数；其它非空参数会打印用法并以退出码 2 退出：
```
usage: go_ultra [reset-admin-password]
```
