# go_ultra 技术栈与架构全览

> 学习向文档：逐层拆解本项目用到的每一项技术、选型理由与在代码中的落点。

---

## 整体架构

```
浏览器 (React SPA)
    │  HTTPS (Cloudflare Tunnel)
    ▼
Caddy (反代 + TLS + 速率限制)
    │  /api/* → localhost:8080
    │  /*     → web/dist (静态文件 + SPA fallback)
    ▼
Gin HTTP Server (:8080)
    │  中间件链: RequestID → Logger → Recover → OriginCheck
    │  鉴权中间件: PlayerAuth / AdminAuth
    │  handler → service → db/sqlc → SQLite
    ▼
SQLite (modernc 纯 Go 驱动, WAL 模式)
```

---

## 一、后端 (Go)

### 1.1 语言与工具链

| 技术 | 版本 | 用途 | 落点 |
|------|------|------|------|
| **Go** | 1.22+ (实际 1.26) | 后端语言 | `server/go.mod` |
| **git** | 2.54 | 版本管理 | 仓库根 |
| **sqlc** | 1.27 | 从 SQL 生成类型安全 Go 代码 | `server/sqlc.yaml` → `internal/db/sqlc/` |
| **goose** | 3.22 | 数据库迁移工具 | `internal/db/migrations/` |

**为什么选 Go？** 单体二进制部署 (编译成单个 .exe)，无运行时依赖，Windows 本机直接运行。Go 的并发模型 (`goroutine`) 适合 HTTP 服务，标准库 `testing` 无需额外测试框架。

**sqlc 是什么？** 你手写 `.sql` 文件 (带 `-- name:` 注释)，sqlc 扫描它 + 数据库 schema，自动生成类型安全的 Go 代码 (结构体、查询方法、接口)。编译期保证 SQL 与 Go 类型匹配，避免运行时拼写错误。比 ORM 更透明，比手写 `database/sql` 更安全。

**goose 是什么？** 数据库迁移管理工具。你把建表/改表 SQL 写在编号文件里 (`00001_init.sql`)，goose 追踪哪些已经执行过，支持 `up` (正向迁移) 和 `down` (回滚)。

### 1.2 分层架构

```
cmd/go_ultra/main.go          ← 程序入口，装配所有依赖
    │
internal/
  config/                      ← 配置加载 (DB路径、监听地址、Origin白名单)
    config.go                    Load() 从环境变量读配置
    │
  handler/                     ← HTTP 层 (Gin handler)
    router.go                    NewRouter(Deps) 注册全部路由
    response.go                  respondError() 统一错误响应
    health.go                    GET /api/healthz
    auth.go                      登录/登出/me + admin 鉴权
    player.go                    玩家查询 (列表/详情/历史/对局)
    match.go                     对局录入/列表/软删除/恢复
    leaderboard.go               排行榜 + 多人对比
    │
  middleware/                   ← Gin 中间件
    middleware.go                 RequestID / Logger / Recover
    auth.go                       PlayerAuth / AdminAuth (接口注入)
    csrf.go                       OriginCheck (fail-closed Origin 校验)
    │
  service/                     ← 业务逻辑层
    player.go                     LoginOrCreate / GetStats / 连胜
    match.go                      Record (事务内读-算-写) / History
    leaderboard.go                List (排行榜) / CompareData (对比)
    admin.go                      密码管理 / 会话 / 软删除 / 退避
    convert.go                    时间/类型转换工具
    │
  db/                          ← 数据库连接层
    db.go                         New(path) 打开 SQLite + 设 PRAGMA + 跑迁移
    migrations/                   00001_init.sql (goose 迁移)
    sqlc/                         自动生成的类型安全查询代码
    │
  domain/                      ← 纯领域模型 (零外部依赖)
    types.go                      Player / Stats / Match 结构体
    elo.go                        ExpectedScore / ComputeDelta (Elo算法)
    rank.go                       Dan(段位映射) / RankFloor
    errors.go                     *Error 类型 + 预定义哨兵错误
    │
  session/                     ← 会话工具
    session.go                    NewToken / TTL常量 / Cookie名常量
```

**分层原则（依赖方向）**：`handler → service → db`，`domain` 被所有人引用但不引用任何人。每层只知道自己下一层的**接口**，不依赖具体实现。

### 1.3 Web 框架: Gin

| 概念 | 说明 | 落点 |
|------|------|------|
| `gin.Engine` | 路由器实例 | `handler/router.go` `NewRouter()` |
| `gin.Context` | 请求上下文 (携带请求/响应/中间件状态) | 每个 handler 的第一个参数 |
| `c.JSON()` | 写 JSON 响应 | handler 各处 |
| `c.ShouldBindJSON()` | JSON body → 结构体 | `auth.go` `handleLogin` |
| `c.Param()` | URL 路径参数 (`:id`, `:username`) | `player.go` `handleGetPlayer` |
| `c.Query()` | URL 查询参数 (`?limit=20`) | `match.go` `parseLimitOffset` |
| `c.Set()`/`c.Get()` | 中间件间共享值 (logger/playerID/requestID) | middleware 注入，handler 读取 |
| `c.AbortWithStatusJSON()` | 中断链并写错误响应 | `response.go` `respondError` |
| `r.Group()` | 路由分组 + 组级中间件 | `router.go` `api := r.Group("/api")` |
| `r.Use()` | 注册中间件 | `router.go` 全局中间件链 |

**为什么选 Gin？** Go 生态中最成熟的 HTTP 框架之一，性能高，API 简洁，中间件模型清晰。

### 1.4 数据库: SQLite + modernc

| 概念 | 说明 | 落点 |
|------|------|------|
| **modernc.org/sqlite** | 纯 Go 的 SQLite 驱动 (无 CGO) | `db.go` `import _ "modernc.org/sqlite"` |
| `sql.Open("sqlite", dsn)` | 打开连接 | `db.go` |
| `_pragma=foreign_keys(1)` | 启用外键约束 | DSN 参数 |
| `_pragma=journal_mode(WAL)` | WAL (Write-Ahead Log) 模式 | DSN 参数 |
| `_pragma=busy_timeout(5000)` | 写冲突等待 5 秒 | DSN 参数 |
| `_txlock=immediate` | BEGIN IMMEDIATE (防止并发写死锁) | DSN 参数 |
| `*sql.DB` | 数据库连接池 | `db.New()` 返回 |
| `db.BeginTx()` | 开启事务 | `match.go` `Record` |
| `sqlc.Queries.WithTx()` | 在事务内执行 sqlc 查询 | `match.go` `Record` |

**为什么 SQLite？** 朋友圈 <100 人，单文件数据库零运维。modernc 纯 Go 驱动在 Windows 下零 C 依赖，编译/部署无额外工具链。

**WAL 模式是什么？** 默认 SQLite 用 rollback journal (写时锁全库)。WAL 模式下读写可并发——读者不阻塞写者，写者不阻塞读者。但只能有一个写者 (用 `_txlock=immediate` + `busy_timeout` 串行化)。

### 1.5 sqlc 代码生成

**输入**: `.sql` 文件
```sql
-- name: GetPlayerByID :one
SELECT * FROM players WHERE id = ?;
```

**输出**: 
- `models.go` — 表结构对应的 Go struct (字段类型从 SQL 列类型推断)
- `players.sql.go` — `GetPlayerByID(ctx, id)` 方法，参数类型安全检查
- `querier.go` — `Querier` 接口 (所有查询方法的集合，`emit_interface: true`)

**命名规则**:
- `:one` → 返回单个行结构体
- `:many` → 返回 `[]行结构体`
- `:exec` → 无返回值
- 参数 → 自动生成 `<Name>Params` 结构体
- 自定义投影 (如 `SELECT a, b`) → 自动生成 `<Name>Row` 结构体

### 1.6 项目工程约定

| 约定 | 说明 |
|------|------|
| `internal/` | Go 的包可见性墙——外部模块无法 import `go_ultra/internal/*` |
| `go_ultra` module | `server/go.mod` 的 module 名，所有 import 以 `go_ultra/` 开头 |
| embed | `//go:embed migrations/*.sql` 把迁移 SQL 编译进二进制，无需运行时找文件 |
| `t.TempDir()` | 测试中自动创建+清理临时目录 |
| `*_test.go` | 标准测试文件命名 |

---

## 二、前端 (React + TypeScript)

### 2.1 语言与工具链

| 技术 | 版本 | 用途 | 落点 |
|------|------|------|------|
| **Node.js** | 24.15 | JS 运行时 | 全局安装 |
| **pnpm** | 11.9 | 包管理器 (快、省磁盘) | `web/` 下所有 `pnpm` 命令 |
| **Vite** | 8.1 | 开发服务器 + 生产构建 | `web/vite.config.ts` |
| **TypeScript** | 6.0 | 类型系统 | `web/tsconfig.json` |
| **Vitest** | 4.1 | 测试框架 (Vite 原生) | `web/vite.config.ts` test 配置 |
| **React** | 19.2 | UI 框架 | `web/src/` |
| **Tailwind CSS** | 3.4 | 原子化 CSS 框架 | `web/tailwind.config.ts` |
| **shadcn/ui** | new-york style | 组件库 (复制进项目) | `web/src/components/ui/` |

**为什么 pnpm？** 比 npm 快 2-3x，严格的依赖隔离 (幽灵依赖问题被消除)，磁盘空间省 (硬链接共享)。

**为什么 Vite？** 开发时用原生 ESM (浏览器直接加载 .tsx，无需打包)，热更新毫秒级。生产构建用 Rollup，产物小。

**为什么 Tailwind？** 原子化 CSS——类名即样式 (`flex items-center gap-2`)，无需切 HTML/CSS 文件。搭配 shadcn/ui 的 CSS 变量实现深色主题。

**为什么 shadcn/ui？** "复制进项目" 而非 npm 安装——组件源码在你的 `components/ui/` 下，可随意修改。基于 Radix UI (无障碍 headless 组件) + Tailwind 样式。

### 2.2 前端目录结构

```
web/
  src/
    api/                  ← 后端 API 调用层
      types.ts              TS 类型 (snake_case，对齐后端 JSON)
      client.ts             axios 实例 + 错误拦截 + ApiError
      players.ts            玩家相关请求 (login/logout/getMe/...)
      matches.ts            对局/排行榜/对比请求
      admin.ts              管理员请求
    lib/                  ← 纯函数库 (无 React 依赖)
      rank.ts               danOf / danLabel / danColor
      elo-preview.ts        expectedScore / computeDelta / previewMatch
      utils.ts              cn() — Tailwind 类名合并
      echarts-theme.ts      5 色调色板 + 轴颜色 + 段位边界线
      __fixtures__/         rank_cases.csv (与后端共享)
    hooks/                ← React Hooks
      useAuth.ts            React Query 封装 /api/me
    components/           ← React 组件
      ui/                   12 个 shadcn 基础组件
      Layout.tsx            顶部导航 + 内容容器
      AuthGuard.tsx         未登录→重定向 /login
      AdminGuard.tsx        未授权→密码框 (支持 RATE_LIMITED toast)
      PlayerOverview.tsx    页面布局 B (曲线+统计+最近对局+录入)
      RatingChart.tsx       单玩家评分曲线 (ECharts + 段位参考线)
      CompareChart.tsx      多玩家曲线对比
      RankBadge.tsx         段位徽章 (颜色+文字)
      MatchTable.tsx        对局列表表格
      PlayerCombobox.tsx    玩家搜索选择器
      SubmitMatchDialog.tsx 录入弹窗 (表单校验 + 实时 Elo 预览)
    pages/                ← 路由页面
      Login.tsx
      Dashboard.tsx (我的主页)
      PlayerDetail.tsx (玩家详情)
      Leaderboard.tsx (排行榜)
      Compare.tsx (多人对比)
      Admin.tsx (管理员面板)
    test/
      setup.ts             vitest 全局 setup (jest-dom + cleanup + polyfills)
    App.tsx               ← 路由表 (react-router v6)
    main.tsx              ← 入口 (ReactDOM + QueryClient + BrowserRouter + Toaster)
    index.css             ← Tailwind 指令 + shadcn 深色 CSS 变量
```

### 2.3 关键技术选型详解

#### React Query (@tanstack/react-query v5)

| 概念 | 说明 | 落点 |
|------|------|------|
| `useQuery` | 声明式数据获取 (自动缓存/重取/loading/error状态) | `useAuth.ts`, `Leaderboard.tsx` |
| `useMutation` | 写操作 (loading/error/success 状态) | `Login.tsx`, `Admin.tsx` |
| `queryKey` | 缓存键 (`["me"]`, `["player", username]`) | 各处 |
| `staleTime` | 数据视为新鲜的时间 (30s 内不重取) | 各处 |
| `invalidateQueries` | 主动标记缓存失效 (触发重取) | 登录/录入后刷新 |
| `QueryClientProvider` | 全局 Provider | `main.tsx` |

**为什么 React Query？** 手写 `useEffect` + `useState` 管理异步数据极其繁琐 (loading/error/data/refetch)。React Query 把这些状态统一管理，自带缓存、去重、后台刷新。v5 比 v4 简化了 API。

#### react-hook-form + zod

| 概念 | 说明 | 落点 |
|------|------|------|
| `useForm` | 表单状态管理 (register/handleSubmit/errors) | `Login.tsx`, `SubmitMatchDialog.tsx` |
| `zodResolver` | 把 zod schema 转为 react-hook-form 校验 | `Login.tsx` |
| `z.object({...})` | 定义字段约束 (min/max/regex) | `Login.tsx`, `SubmitMatchDialog.tsx` |

**为什么 react-hook-form？** 非受控组件 (用 ref 而非 state)，不触发重渲染，性能好。搭配 zod 做类型安全的校验。

#### ECharts (echarts-for-react)

| 概念 | 说明 | 落点 |
|------|------|------|
| `echarts-for-react` | ECharts 的 React 封装 (`<ReactECharts option={...} />`) | `RatingChart.tsx`, `CompareChart.tsx` |
| `option` | ECharts 配置对象 (xAxis/yAxis/series/tooltip/...) | 组件内构建 |
| `EChartsOption` | TypeScript 类型 | `import type { EChartsOption } from "echarts"` |
| 暗色主题 | 通过配置 `backgroundColor/axisLine/textStyle` 实现 | `echarts-theme.ts` |
| `markLine` | 水平参考线 (段位边界) | `RatingChart.tsx` |

#### react-router-dom v6

| 概念 | 说明 | 落点 |
|------|------|------|
| `<BrowserRouter>` | 根组件 | `main.tsx` |
| `<Routes>` + `<Route>` | 路由表 | `App.tsx` |
| `useNavigate()` | 编程式跳转 | `Layout.tsx`, `Login.tsx` |
| `useParams()` | 读 URL 路径参数 | `PlayerDetail.tsx` |
| `useSearchParams()` | 读/写 URL query string | `Compare.tsx` |
| `<Navigate to="..." replace>` | 重定向 | `AuthGuard.tsx`, `RootRedirect` |

---

## 三、部署层

| 技术 | 用途 | 落点 |
|------|------|------|
| **Caddy** | 反向代理 + 自动 HTTPS + 速率限制 + 静态文件 | `Caddyfile` |
| **caddy-ratelimit** | Caddy 插件，按 IP 限速 | `Caddyfile` `rate_limit` 指令 |
| **Cloudflare Tunnel** | 内网穿透 (无需公网 IP) | `start.bat` |
| **Caddyfile** | Caddy 配置 (类似 nginx.conf) | 仓库根 |
| **start.bat** | 启后端→探活→启 Caddy→启 cloudflared | 仓库根 |
| **stop.bat** | taskkill 三个进程 | 仓库根 |
| **build.bat** | pnpm build + go build | `scripts/` |
| **dev.bat** | 并行 vite dev + go run | `scripts/` |

**为什么 Caddy？** 自动 HTTPS (Let's Encrypt)，配置文件比 nginx 简洁一个数量级，原生支持 API 速率限制 (通过插件)。单文件二进制，Windows 部署无异于 Linux。

**为什么 Cloudflare Tunnel？** 没有公网 IP 的家用宽带也能对外暴露服务。Cloudflare 边缘 TLS 终结 + DDoS 防护，内网段用自签证书 (`tls internal`)。

---

## 四、测试策略

| 层级 | 框架 | 覆盖率 | 测试文件 |
|------|------|--------|---------|
| domain | Go `testing` (table-driven) | 100% | `_test.go` |
| db | Go `testing` (集成测试，真实 SQLite) | 65% | `_test.go` |
| service | Go `testing` (真实 SQLite 集成) | 80.6% | `_test.go` |
| handler | Go `testing` + `httptest` (真实 router+sqlite) | 75.4% | `_test.go` |
| 前端 lib | Vitest (纯函数) | 100% | `*.test.ts` |
| 前端组件 | Vitest + Testing Library (jsdom) | 83.3% | `*.test.tsx` |

**测试原则**: 后端不 mock DB (用 `t.TempDir()` 开临时 SQLite 文件走真实迁移)，前端不 mock API 调用层 (ECharts 因 jsdom 无 Canvas 才 mock)。

---

## 五、关键设计决策速查

| 决策 | 理由 |
|------|------|
| `domain.Error` 不用 `Unwrap()`/`Is()` | service 直接返回 `*domain.Error`，handler 用 `errors.As` 断言 |
| CSV fixture 前后端共享且 LF-pinned | `core.autocrlf=true` 会 CRLF 化，`.gitattributes eol=lf` 锁定 |
| `_txlock=immediate` | modernc 忽略 `sql.LevelSerializable`；不加则并发写死锁 |
| 全局管理员退避 (非按 IP) | 只有一个管理员；全局防 IP 轮换绕过 |
| 纯前端 Elo 预览 | 与后端同公式，录入弹窗实时计算无需网络请求 |
| 段位五色 `danColor` | 灰/蓝/紫/金/红 → 视觉直观 (卡牌游戏式分段) |
| Dark-only (无亮色) | 深色主题是默认且唯一主题，CSS 变量用 HSL |
| snake_case JSON | `emit_json_tags` 自动生成，前后端对齐零转换 |

---

## 六、快速启动 (开发/部署)

```bat
rem 开发模式 (前后端并行热更新)
scripts\dev.bat

rem 生产构建
scripts\build.bat

rem 启动服务 (需先 build + 配好 Cloudflare Tunnel)
start.bat

rem 停止服务
stop.bat

rem 重置管理员密码
cd server && go_ultra.exe reset-admin-password
```

---

> 更多细节见 `docs/superpowers/specs/2026-06-25-go-ultra-design.md` (设计规范)
> 和 `docs/superpowers/plans/2026-06-25-go-ultra-implementation.md` (实施计划)
