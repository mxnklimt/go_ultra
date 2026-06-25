# go_ultra — 设计文档

**日期**：2026-06-25
**版本**：v1.0（MVP）
**状态**：已经过头脑风暴，待审阅

---

## 1. 项目概述

### 1.1 目标

通用 1v1 竞技游戏等级分系统，供朋友圈 &lt; 100 人使用。系统提供：

- 玩家个人主页：等级分历史曲线、最近对局、统计
- 排行榜：所有玩家按当前等级分排名
- 对局录入：玩家自助录入 1v1 对局结果，系统自动用 Elo 公式更新双方等级分
- 多人曲线对比：在同一张图叠加多名玩家的历史曲线，并列出两两交手记录
- 玩家详情：查看任意玩家的曲线和历史
- 管理员功能：删除（软删除）和恢复对局

### 1.2 非目标

- 不支持团队/多人对局（仅 1v1）
- 一个系统实例只服务一个游戏；不支持单实例多游戏并行
- 不支持密码/邮箱注册（信任模式）
- MVP 阶段不做实时推送（用户手动刷新即可）
- 不做移动端原生 App（响应式 web 即可）

### 1.3 规模与部署

- 玩家数：&lt; 100
- 对局频率：每天数十局以下
- 部署形态：用户本人 Windows 电脑 + Caddy + Cloudflare Tunnel 暴露公网

### 1.4 通用性与适配约束（重要）

**本系统的定位是通用 1v1 竞技游戏等级分系统**，不绑定任何特定竞技项目。系统的所有核心机制 —— Elo 计算、段位映射、对局录入、历史曲线、多人对比、排行榜、软删除、管理员控制台 —— 都不假设任何游戏特定语义。

部署到任何具体 1v1 竞技场景只需修改以下"表层"，**不需要触碰任何业务逻辑**：

| 适配点 | 默认值 / 范围 | 适配方式 |
|---|---|---|
| 项目名 / 二进制名 | `go_ultra` | 改 `go.mod` 模块名、`README`、`start.bat` 命名、Cloudflare Tunnel 域名 |
| UI 文案 / 术语 | "对局"、"段位"、"等级分" 等中性词 | 修改前端文案常量或 i18n 字典 |
| **K 因子** | 默认 16；可按所在场景的节奏与样本量调整 | 改 `domain/elo.go` 常量 |
| **起始分** | 默认 1500 | 改 `domain/elo.go` 常量 |
| **段位映射** | 默认段 1–9，等宽 200 分（详见 §4.3）| 替换 `domain/rank.go` 中的边界表与段位标签（如 "段 1 / Bronze / 新手 / Rank I"）|
| 段位标签 | 默认 "段 1 / 段 2 … 段 9"（与数据库 `dan` 字段对应）| 改为任何分层标签 |

**为支撑"改常量即可换场景"的目标，实现时遵循以下约束：**

- 所有可变常量（K 因子、起始分、段位边界、段位标签）**集中在 `server/internal/domain/elo.go` 和 `server/internal/domain/rank.go`**，禁止散落到 service/handler 层
- 前端段位映射 `web/src/lib/rank.ts` 与后端共享 **同一份段位常量 fixture**（参见 §9.2 CSV fixture）
- 任何 UI 文案（"对局"、"段位"、"等级分"等）不在业务逻辑或 SQL 里硬编码；通过前端文案常量或 i18n 暴露
- 数据库表名、字段名、API 路径用中性通用术语（`matches` / `players` / `rating` / `dan`），**不出现任何具体游戏的专有词汇**
- spec、README、源码注释中**禁止**援引具体游戏作为"举例"或"参照"（避免引入语义偏向）

**通用性范围之外**（不在"改常量即可"的承诺之内）：

- 团队对战、多人混战、排名赛、积分制（保持 1v1 假设；改这些需要数据模型与算法改写）
- 单实例多游戏并行（需要外层 `game_id` 字段；MVP 不实现）
- 段位之外的分层（如赛季、分区、地区）

实施每个阶段都要回答："这个改动放在 domain/* 中是否能让换场景时只改这一处？" —— 如果不行，要么把它移到 domain，要么文档化为"场景专有"。

---

## 2. 关键业务决策

| 决策点 | 选择 | 说明 |
|---|---|---|
| 积分算法 | 经典 Elo，K = 16（默认） | K 反映分数变动幅度；越稳态越小，越快节奏越大 |
| 起始分 | 1500 | 默认分层中段（段 3 区间内） |
| 期望胜率公式 | `E_A = 1 / (1 + 10^((R_B − R_A) / 400))` | 标准 Elo |
| 等级分上下限 | 无 | 自然收敛 |
| 显示精度 | 整数（四舍五入） | |
| 段位映射 | 每 200 分一段，1050 以下不显示段位 | 见 §4.3 |
| 对局录入 | 任一玩家自助录入，提交即生效 | 无需对方确认 |
| 登录方式 | 仅用户名（无密码） | 朋友圈信任模式 |
| 用户资料 | 仅用户名 | |
| 删除对局语义 | D1b：软删除，**不重算**后续局 | 后续局快照分保留不变 |
| 删除权限 | 仅管理员 | |
| 管理员认证 | bcrypt 哈希密码（首次启动随机生成） | 30 分钟会话有效 |
| 数据保留 | 永久 | 不自动备份 |

---

## 3. 系统架构

### 3.1 拓扑

```
朋友 (浏览器, HTTPS)
       │
       ▼
┌──────────────────────┐
│ Cloudflare Tunnel    │   cloudflared 反代 → 用户机器 :443
└──────────┬───────────┘
           ▼
┌──────────────────────┐
│ Caddy (本机:443)     │   自动 TLS, 反代路由：
│                      │     /api/*  → :8080  (Go 后端)
│                      │     /*      → web/dist (静态)
└──────────┬───────────┘
           ├──────────────┐
           ▼              ▼
   ┌──────────────┐  ┌──────────────┐
   │ go_ultra.exe │  │ web/dist     │
   │ Gin :8080    │  │ (静态资源)   │
   └──────┬───────┘  └──────────────┘
          │
          ▼
   ┌──────────────┐
   │ SQLite (.db) │   本机单文件
   └──────────────┘
```

### 3.2 进程组成

| 进程 | 作用 |
|---|---|
| `cloudflared` | Cloudflare Tunnel，安装为 Windows 服务自启 |
| `caddy` | 反代 + 自动 TLS，前端静态文件服务，速率限制 |
| `go_ultra.exe` | Gin API 服务，访问 SQLite |

### 3.3 开发模式

- 前端 `vite dev` 监听 :5173，通过 `vite.config.ts` proxy 把 `/api` 转发到 :8080
- 后端 `go run` 监听 :8080
- 同源生产部署不需要 CORS；开发期由 Vite proxy 解决

---

## 4. 数据模型

### 4.1 表结构

```sql
CREATE TABLE players (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE COLLATE NOCASE,
    rating          INTEGER NOT NULL DEFAULT 1500,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_players_rating ON players(rating DESC);

CREATE TABLE matches (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    winner_id       INTEGER NOT NULL REFERENCES players(id),
    loser_id        INTEGER NOT NULL REFERENCES players(id),
    submitter_id    INTEGER NOT NULL REFERENCES players(id),

    winner_rating_before  INTEGER NOT NULL,
    loser_rating_before   INTEGER NOT NULL,
    winner_rating_after   INTEGER NOT NULL,
    loser_rating_after    INTEGER NOT NULL,
    winner_delta          INTEGER NOT NULL,
    loser_delta           INTEGER NOT NULL,

    played_at       TIMESTAMP NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    deleted_at      TIMESTAMP,
    deleted_by      INTEGER REFERENCES players(id),

    CHECK (winner_id != loser_id)
);
CREATE INDEX idx_matches_winner ON matches(winner_id, played_at DESC);
CREATE INDEX idx_matches_loser  ON matches(loser_id,  played_at DESC);
CREATE INDEX idx_matches_played ON matches(played_at DESC);
CREATE INDEX idx_matches_active ON matches(deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE sessions (
    token       TEXT PRIMARY KEY,
    player_id   INTEGER NOT NULL REFERENCES players(id),
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP NOT NULL
);
CREATE INDEX idx_sessions_player ON sessions(player_id);

CREATE TABLE admin_sessions (
    token       TEXT PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP NOT NULL
);

CREATE TABLE settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL
);
-- 首次启动插入：('admin_password_hash', bcrypt(随机生成密码))
```

### 4.2 关键设计取舍

- **快照式存储**：每条 `matches` 记录都包含赛前/赛后双方分数，不依赖 `players.rating` 反推。
- **`players.rating`** 只是"最新值缓存"，方便排行榜 O(1) 查询。
- **D1b 删除语义**：软删除一局只设 `deleted_at`；后续对局快照分**保持不变**；当前 `players.rating` 也不变。
- **没有积分历史快照表**：matches 本身就是积分变化的事件流。
- **未达起始段处理**：rating &lt; 1050 时段位映射函数返回 `dan = 0`（"段 0"，UI 不显示段位徽章），不在数据库存"段位"字段。
- **预留"假如重算"工具**：MVP 不实现；管理员未来可一键基于所有未删除对局重算所有 rating，消除 D1b 的"幽灵分数"。

### 4.3 段位映射

| 段位 | 中分（标志值）| 区间（含下不含上） |
|---|---|---|
| 0 | — | rating &lt; 1050（不显示段位） |
| 1 | 1100 | [1050, 1200) |
| 2 | 1300 | [1200, 1400) |
| 3 | 1500 | [1400, 1600) |
| 4 | 1700 | [1600, 1800) |
| 5 | 1900 | [1800, 2000) |
| 6 | 2100 | [2000, 2200) |
| 7 | 2300 | [2200, 2400) |
| 8 | 2500 | [2400, 2600) |
| 9 | 2600+ | rating ≥ 2600（无上限）|

实现（伪代码）：

```
function dan(rating):
    if rating < 1050:        return 0          # 段 0：UI 不显示段位徽章
    if rating >= 2600:       return 9          # 段 9：顶端开区间
    return (rating - 1050) // 200 + 1          # 段 1..8
```

验证：1049→0、1050→1、1199→1、1200→2、1599→3、1600→4、2599→8、2600→9。

段内细分（下/中/上）作为 tooltip / 详情页浮层附加信息，主界面只显示段位即可：下 = 段位起点；中 = 起点 + 67；上 = 起点 + 133（按段宽 200 三等分）。段 0 与段 9 不显示亚分级。

### 4.4 Elo 计算（伪代码）

```
function record_match(winner_id, loser_id, played_at, submitter_id):
    BEGIN IMMEDIATE TRANSACTION
    A = players[winner_id].rating
    B = players[loser_id].rating
    E_A = 1 / (1 + 10^((B − A) / 400))
    delta = round(16 * (1 − E_A))
    winner_after = A + delta
    loser_after  = B − delta
    INSERT INTO matches (...) VALUES (
      winner_id, loser_id, submitter_id,
      A, B, winner_after, loser_after,
      +delta, −delta,
      played_at, NOW(), NULL, NULL
    )
    UPDATE players SET rating = winner_after WHERE id = winner_id
    UPDATE players SET rating = loser_after  WHERE id = loser_id
    COMMIT
```

零和性：`winner_delta + loser_delta == 0`，避免分数漂移。

### 4.5 历史曲线查询

```sql
SELECT played_at,
       CASE WHEN winner_id = ? THEN winner_rating_after
            ELSE loser_rating_after END AS rating
FROM matches
WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL
ORDER BY played_at ASC;
```

前端 prepend `(player.created_at, 1500)` 作为曲线起点。

### 4.6 迁移管理

使用 [goose](https://github.com/pressly/goose)，迁移文件位于 `server/internal/db/migrations/`。后端启动时自动执行未应用的迁移。每个迁移都有 down 脚本以备回滚。

---

## 5. 目录结构

```
go_ultra/
├── README.md
├── start.bat                    # 一键启动 caddy + go + cloudflared
├── stop.bat
├── Caddyfile
├── docs/superpowers/specs/      # 设计文档
│
├── server/                      # Go 后端
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/go_ultra/main.go
│   ├── internal/
│   │   ├── config/              # 配置加载
│   │   ├── db/
│   │   │   ├── migrations/      # *.sql goose 迁移
│   │   │   └── sqlc/            # sqlc 生成
│   │   ├── domain/              # 纯业务模型 + Elo 算法
│   │   │   ├── elo.go
│   │   │   ├── rank.go
│   │   │   ├── errors.go
│   │   │   └── types.go
│   │   ├── service/             # 业务服务层
│   │   │   ├── player.go
│   │   │   ├── match.go
│   │   │   ├── leaderboard.go
│   │   │   └── admin.go
│   │   ├── handler/             # Gin handler
│   │   │   ├── router.go
│   │   │   ├── auth.go
│   │   │   ├── player.go
│   │   │   ├── match.go
│   │   │   ├── leaderboard.go
│   │   │   └── admin.go
│   │   ├── middleware/          # auth, request_id, recover, log
│   │   └── session/             # cookie session 实现
│   ├── queries/                 # sqlc 输入 SQL
│   └── sqlc.yaml
│
├── web/                         # 前端
│   ├── package.json
│   ├── pnpm-lock.yaml
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   ├── public/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx              # 路由
│   │   ├── api/
│   │   │   ├── client.ts        # axios + interceptor
│   │   │   ├── players.ts
│   │   │   ├── matches.ts
│   │   │   ├── admin.ts
│   │   │   └── types.ts
│   │   ├── components/
│   │   │   ├── ui/              # shadcn/ui
│   │   │   ├── Layout.tsx
│   │   │   ├── RankBadge.tsx
│   │   │   ├── RatingChart.tsx
│   │   │   ├── CompareChart.tsx
│   │   │   ├── MatchTable.tsx
│   │   │   ├── SubmitMatchDialog.tsx
│   │   │   ├── PlayerCombobox.tsx
│   │   │   ├── AuthGuard.tsx
│   │   │   └── AdminGuard.tsx
│   │   ├── pages/
│   │   │   ├── Login.tsx
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Leaderboard.tsx
│   │   │   ├── PlayerDetail.tsx
│   │   │   ├── Compare.tsx
│   │   │   └── Admin.tsx
│   │   ├── hooks/               # useAuth, usePlayer ...
│   │   ├── lib/
│   │   │   ├── rank.ts          # 前端段位映射镜像
│   │   │   └── elo-preview.ts
│   │   └── styles/
│   └── dist/                    # 构建产物
│
└── scripts/
    ├── dev.bat                  # vite dev + go run 并行
    └── build.bat                # pnpm build + go build
```

---

## 6. API 设计

所有路径前缀 `/api`，JSON 请求/响应。Session 通过 HttpOnly Cookie 传递：
- 玩家会话：`go_ultra_session`
- 管理员会话：`go_ultra_admin`

### 6.1 端点

#### 鉴权

```
POST   /api/login              { "username": "alice" }
                               → 200 / 201 { player: {...} }, Set-Cookie
                               用户名不存在则自动创建（= 隐式注册）

POST   /api/logout             → 204
GET    /api/me                 → { player: {...} }

POST   /api/admin/login        { "password": "..." }
                               → 200 { expires_at: "..." }, Set-Cookie
POST   /api/admin/logout       → 204
GET    /api/admin/status       → { authed: bool, expires_at?: "..." }
```

#### 玩家

```
GET    /api/players
         → [{ id, username, rating, dan, games_played, win_rate, ... }]

GET    /api/players/:username
         → { id, username, rating, dan, created_at,
             stats: { wins, losses, win_rate, current_streak, longest_streak } }

GET    /api/players/:username/history?from=&to=
         → [{ played_at, rating }]

GET    /api/players/:username/matches?limit=50&offset=0
         → [{ id, opponent, result, rating_before, rating_after, delta, played_at }]
```

#### 对局

```
POST   /api/matches
         { "opponent_username": "bob",
           "result": "win" | "loss",
           "played_at": "2026-06-25T14:30:00Z" }   // 可选；默认当前时间；无时间限制
         → 201 { id, winner_delta, loser_delta,
                 new_self_rating, new_opponent_rating }

GET    /api/matches?limit=50&offset=0
         → 全局对局流（不含已删除）

DELETE /api/matches/:id        → 204  仅管理员；软删除

GET    /api/admin/matches/deleted          → 已删除对局列表（仅管理员）
POST   /api/admin/matches/:id/restore      → 204 取消软删除（仅管理员）
```

#### 排行榜

```
GET    /api/leaderboard?min_games=0
         → [{ rank, username, rating, dan, games_played, win_rate }, ...]
```

#### 对比

```
GET    /api/compare?usernames=alice,bob,charlie&from=&to=
         → {
             series: [
               { username, color, points: [{ played_at, rating }] },
               ...
             ],
             head_to_head: [
               { a: "alice", b: "bob", a_wins: 5, b_wins: 3 },
               ...
             ]
           }
```

### 6.2 错误响应

统一格式：
```json
{ "error": { "code": "PLAYER_NOT_FOUND", "message": "..." } }
```

| HTTP | code | 场景 |
|---|---|---|
| 400 | INVALID_BODY / INVALID_PARAM | 校验失败 |
| 401 | NOT_AUTHENTICATED | 缺/过期 cookie |
| 403 | ADMIN_REQUIRED | 非管理员访问管理端点 |
| 404 | PLAYER_NOT_FOUND / MATCH_NOT_FOUND | |
| 409 | SELF_MATCH | 自己对自己 |
| 429 | RATE_LIMITED | 速率限制触发 |
| 500 | INTERNAL | 兜底 |

### 6.3 设计原则

- REST 风格，复数路径
- 隐式注册（登录即自动建账号）
- `played_at` 可选，默认当前时间，**无任何时间范围限制**
- DELETE 是软删除，对前端透明（不暴露 `deleted_at`）
- MVP 不提供"更新对局"端点

---

## 7. 前端页面与组件

### 7.1 路由

| 路径 | 页面 |
|---|---|
| `/` | 已登录跳 `/me`，否则 `/login` |
| `/login` | 登录页 |
| `/me` | 自己的主页 |
| `/leaderboard` | 排行榜 |
| `/players/:username` | 玩家详情（复用 `/me` 布局） |
| `/compare?p=a,b,c` | 多人对比 |
| `/submit` | 录入对局（也可在 `/me` 内以弹窗弹出） |
| `/admin` | 管理员面板 |

### 7.2 布局决策（已确认）

| 页面 | 选定布局 |
|---|---|
| 我的主页 | **B**：大曲线 + 右侧栏（统计 + 录入按钮 + 最近对局），曲线占视觉主导 |
| 排行榜 | **A 表格 + 顶部领奖台**：Top 3 用领奖台，4 名往后用紧凑表格 |
| 多人对比 | **B + C 合并**：左侧栏控制玩家选择/显隐/时间 + 右侧大曲线 + 下方"头对头"统计卡片 |
| 录入对局 | **A 标准表单**：对手 → 结果 → 时间 → Elo 预览 → 提交 |
| 玩家详情 | 复用"我的主页"布局，右上加"📊 对比"按钮 |
| 管理员面板 | 已删除对局列表（表格风格），可恢复 |

### 7.3 顶部导航

```
🎯 go_ultra    我的  排行榜  对比  录入对局                  alice ▾
```
右上角用户菜单：个人主页 / 登出 / （管理员）管理面板。

### 7.4 核心组件

| 组件 | 作用 |
|---|---|
| `Layout` | 顶部导航 + 内容容器 |
| `RankBadge` | 分数 → 段位徽章（颜色按段位渐变） |
| `RatingChart` | 单玩家曲线（ECharts），含段位水平线 |
| `CompareChart` | 多玩家曲线叠加（ECharts），同步 tooltip |
| `MatchTable` | 对局列表表格 |
| `SubmitMatchDialog` | 录入弹窗 + Elo 预览 |
| `PlayerCombobox` | 玩家选择器（自动补全） |
| `AuthGuard` / `AdminGuard` | 路由守卫 |

### 7.5 视觉风格

- **shadcn/ui** 默认风格：黑白灰 + 圆角 + 微妙阴影，深色主题
- **段位徽章配色**：段 0 灰、段 1-3 蓝、段 4-6 紫、段 7-8 金、段 9 红
- **ECharts** dark 主题；自定义 5 色板适合 5 人以下对比，超 5 用渐变色环
- 桌面优先，平板可用，手机仅"能看"

### 7.6 数据获取策略

- 用 **React Query (@tanstack/react-query)** 缓存与后台刷新
- 排行榜 / 详情页 `staleTime: 30s`
- 录入对局后 `invalidateQueries(['leaderboard', 'me', 'players'])` 强制刷新
- 没有 WebSocket / SSE

---

## 8. 错误处理、日志、安全

### 8.1 后端

**错误处理**
- 自定义 `domain.Error` 类型（Code / Message / Status / Cause）
- handler 层中间件统一捕获 `AppError → JSON`，`panic → 500 + 日志`
- service 层只抛 `AppError`，不组装 HTTP 响应

**输入校验**
- 用 `go-playground/validator`
- 校验失败 → 400 + 字段清单
- 用户名：3-32 字符，允许中英文/数字/下划线，trim 后非空

**日志**
- 用 `zerolog`，JSON 格式
- stdout + `logs/server.log`（按天滚动，保留 7 天）
- 每 HTTP 请求一行：time / level / request_id / player_id / method / path / status / latency_ms / error

**并发安全**
- 录入对局使用 `BEGIN IMMEDIATE` 事务
- 没有显式锁，依赖 DB 事务

**安全**

| 风险 | 缓解 |
|---|---|
| Cookie 劫持 | `HttpOnly` + `Secure` + `SameSite=Lax`；Cloudflare Tunnel 强制 HTTPS |
| CSRF | SameSite + 所有 POST/DELETE 校验 `Origin` 头 |
| 管理员密码暴力穷举 | 错误密码指数退避：失败次数 N → 锁定 `2^N` 秒（封顶 1 小时） |
| SQL 注入 | sqlc 类型安全查询 |
| XSS | React 默认转义；ECharts 不渲染 HTML 文本 |
| 用户名冒充 | **已知接受风险**（朋友圈信任模式） |
| 拒绝服务 | Caddy 每 IP 60 req/min 速率限制 |

### 8.2 前端

- axios interceptor 统一处理 401 / 错误格式
- toast 提示（shadcn/ui `Toaster`）
- `AuthGuard` / `AdminGuard` 路由守卫；管理员页未授权 → 弹密码框
- 表单用 react-hook-form + zod
- 全局 ErrorBoundary 兜底

### 8.3 部署配置

**Caddyfile**

```
:443 {
    tls internal

    handle /api/* {
        reverse_proxy localhost:8080
    }

    handle {
        root * E:/go_ultra/web/dist
        try_files {path} /index.html
        file_server
    }

    @api path /api/*
    rate_limit @api 60r/m

    log {
        output file E:/go_ultra/logs/caddy.log
    }
}
```

**Cloudflare Tunnel**
- `cloudflared` 安装为 Windows 服务自启
- 在 Cloudflare Zero Trust 控制台创建 Tunnel 拿 token
- 公开域名由 Cloudflare 自动签发证书

**`start.bat`**

```batch
@echo off
echo Starting go_ultra services...
start "go_ultra-api" /B "%~dp0server\go_ultra.exe"
timeout /t 2 /nobreak
start "caddy" /B caddy run --config "%~dp0Caddyfile"
start "tunnel" /B cloudflared tunnel run go-ultra
echo All services started.
```

---

## 9. 测试策略

### 9.1 后端

**单元测试**（目标 100% 行 + 分支覆盖）：
- `domain/elo.go`：边界（分差 0/+400/-400/极端）、K 因子、零和性
- `domain/rank.go`：段位映射边界（1049/1050/1199/1200/2599/2600）
- `domain/errors.go`：类型断言、Status 映射

**集成测试**（用 `:memory:` SQLite）：
- 录入 100 局后两人分数总和守恒
- 录入对局后排行榜位置正确更新
- 软删除后普通查询不返回，`include_deleted` 仍返回
- 管理员恢复对局后查询恢复
- 登录创建用户、二次登录复用、session 过期

**HTTP 测试**：
- 错误响应格式一致性
- 鉴权 middleware
- 管理员路由的双重鉴权

### 9.2 前端

**单元测试**（Vitest）：
- `lib/rank.ts`：段位映射，与后端共享相同 CSV fixture
- `lib/api/client.ts`：错误格式解析、interceptor
- `lib/elo-preview.ts`：录入预览函数

**组件测试**（@testing-library/react）：
- `RankBadge`：不同分数渲染颜色/文本
- `RatingChart`：传入 mock 渲染 ECharts 容器
- `SubmitMatchDialog`：选对手 + 结果 → 预览正确 → 提交触发 API

**E2E（可选，Playwright）** — MVP 不强求：
- 注册 → 登录 → 录入 → 排行榜
- 多人对比加 3 玩家 → 看到 3 条曲线
- 管理员删除 → 列表少一条 → 恢复

### 9.3 命令

```bash
cd server && go test ./...
cd server && go test -cover ./...
cd server && go test -race ./...

cd web && pnpm test
cd web && pnpm test --coverage
```

### 9.4 目标覆盖率

| 层 | 目标 |
|---|---|
| domain/* | 100% |
| service/* | ≥ 80% |
| handler/* | ≥ 70% |
| 前端 lib/* | ≥ 80% |
| 前端 components/* | ≥ 50% |

---

## 10. 风险与未决问题

| 风险 / 问题 | 影响 | 应对 |
|---|---|---|
| Windows 电脑关机/睡眠 → 服务挂掉 | 朋友访问失败 | 接受 — 朋友圈使用，开机才提供服务 |
| 用户名冒充（信任模式） | 任何人可冒充任何已知用户名 | 已知接受 — 朋友圈使用，靠社交信任 |
| D1b "幽灵分数"（删除后曲线不重算） | 视觉上看历史有"凭空消失的点" | 接受 — 未来可加"一键重算"管理功能 |
| 补录很久以前的对局 | 曲线时间轴上出现"过去的点" | 接受 — 符合 D1b 语义 |
| Cloudflare Tunnel 配置一次性成本 | 用户需上手 Tunnel + 域名 | 文档化步骤 |
| SQLite 单文件备份 | 文件损坏即数据丢失 | MVP 不自动备份；用户可手动复制 .db 文件 |

---

## 11. 实施范围摘要

**MVP 必须有**（任务 #5 中确认的功能 A B C E F G）：
- ✅ 玩家自己曲线查看（A）
- ✅ 排行榜（B）
- ✅ 玩家自助录入对局（C）
- ✅ 多玩家曲线对比（E）
- ✅ 衍生统计：胜率、连胜、对手强度等（F）
- ✅ 账户体系：注册（隐式）+ 登录（仅用户名）（G）

**MVP 范围之外**（明确推迟）：
- 对局更新（修改已录入对局）
- 自动备份
- "一键重算"管理工具
- E2E 测试
- 移动端原生 App
- 实时推送（SSE/WebSocket）
- 第三方登录
- 用户头像与个人资料
