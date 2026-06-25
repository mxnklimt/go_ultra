# go_ultra 实施计划 —— 接口契约（唯一真相源）

> 本文件是给"撰写实施计划的各 agent"看的内部参考，不是交付物。计划写完后可删除。
> 每个 agent 必须严格使用下方的名字 / 签名 / 查询名，**不得自创或改名**。

## 全局约定

- Go module path：`go_ultra`（import 形如 `"go_ultra/internal/domain"`）
- Go 版本 1.22+；Gin v1.10；SQLite 驱动 `modernc.org/sqlite`（纯 Go，无 CGO）
- sqlc v1.27；迁移 `pressly/goose` v3；日志 `rs/zerolog`；校验 `go-playground/validator/v10`；bcrypt `golang.org/x/crypto/bcrypt`
- 前端：React 18 + TypeScript + Vite 5 + Tailwind 3 + shadcn/ui + ECharts 5（`echarts-for-react`）+ `@tanstack/react-query` v5 + axios + react-hook-form + zod + react-router-dom v6 + sonner（toast）
- 包管理器：后端 go mod；前端 pnpm
- 所有时间以 UTC 存储为 ISO8601 字符串（RFC3339）。Go 侧 `time.Now().UTC()`。SQLite 列类型写 `TEXT` 存 RFC3339；伪代码里的 `NOW()` 一律实现为 `time.Now().UTC()`。

## domain 层（`go_ultra/internal/domain`）

### types.go
```go
package domain

import "time"

type Player struct {
	ID        int64
	Username  string
	Rating    int
	CreatedAt time.Time
}

type Stats struct {
	Wins          int
	Losses        int
	WinRate       float64
	CurrentStreak int
	LongestStreak int
}

type Match struct {
	ID                 int64
	WinnerID           int64
	LoserID            int64
	SubmitterID        int64
	WinnerRatingBefore int
	LoserRatingBefore  int
	WinnerRatingAfter  int
	LoserRatingAfter   int
	WinnerDelta        int
	LoserDelta         int
	PlayedAt           time.Time
	CreatedAt          time.Time
	DeletedAt          *time.Time
	DeletedBy          *int64
}
```

### elo.go
```
const DefaultRating = 1500
const KFactor = 16
func ExpectedScore(ratingA, ratingB int) float64    // 1/(1+10^((B-A)/400))
func ComputeDelta(winnerRating, loserRating int) int // round(K*(1-E_winner))，用 math.Round（half away from zero）
```
- loser delta = `-ComputeDelta(...)`（零和）

### rank.go
```
const RankFloor = 1050
func Dan(rating int) int
```
实现：
```go
func Dan(rating int) int {
	if rating < RankFloor {
		return 0
	}
	tier := (rating - 800) / 200
	if tier > 9 {
		return 9
	}
	return tier
}
```
边界（必须逐条测试）：1049→0, 1050→1, 1199→1, 1200→2, 1399→2, 1400→3, 1500→3, 1599→3, 1600→4, 2399→7, 2400→8, 2599→8, 2600→9, 5000→9

### errors.go
```go
type Error struct {
	Code    string
	Message string
	Status  int
	Cause   error
}
func (e *Error) Error() string
func (e *Error) WithCause(err error) *Error  // 返回副本，填 Cause
```
预定义：
```go
var ErrPlayerNotFound   = &Error{Code: "PLAYER_NOT_FOUND",  Message: "玩家不存在",       Status: 404}
var ErrMatchNotFound    = &Error{Code: "MATCH_NOT_FOUND",   Message: "对局不存在",       Status: 404}
var ErrSelfMatch        = &Error{Code: "SELF_MATCH",        Message: "不能和自己对局",   Status: 409}
var ErrNotAuthenticated = &Error{Code: "NOT_AUTHENTICATED", Message: "未登录",           Status: 401}
var ErrAdminRequired    = &Error{Code: "ADMIN_REQUIRED",    Message: "需要管理员权限",   Status: 403}
var ErrInvalidBody      = &Error{Code: "INVALID_BODY",      Message: "请求体无效",       Status: 400}
var ErrInvalidParam     = &Error{Code: "INVALID_PARAM",     Message: "参数无效",         Status: 400}
var ErrInternal         = &Error{Code: "INTERNAL",          Message: "服务器内部错误",   Status: 500}
```

## db 层（`go_ultra/internal/db`）

- 迁移文件：`server/internal/db/migrations/00001_init.sql`（goose，含 `-- +goose Up` / `-- +goose Down`）
  - 表结构严格按 spec §4.1，但所有时间列改为 `TEXT`（存 RFC3339）
  - 额外（审阅采纳）matches 增加 3 个 CHECK：
    - `CHECK (winner_rating_after = winner_rating_before + winner_delta)`
    - `CHECK (loser_rating_after  = loser_rating_before  + loser_delta)`
    - `CHECK (winner_delta + loser_delta = 0)`
  - 额外索引：`CREATE INDEX idx_sessions_expires ON sessions(expires_at);`
- `sqlc.yaml`：engine sqlite，queries 目录 `server/queries`，schema 指向 migrations，生成到 `internal/db/sqlc`，`emit_json_tags: true`，`emit_interface: true`（生成 `Querier` 接口）
- 生成包名 `sqlc`，生成 `Queries` 结构体 + `Querier` 接口
- 查询文件 `server/queries/*.sql`，用 `-- name:` 注释。**必须包含且仅用这些查询名**：

| 查询名 | 类型 | 说明 |
|---|---|---|
| `CreatePlayer` | :one | (username, rating) → players row |
| `GetPlayerByID` | :one | |
| `GetPlayerByUsername` | :one | NOCASE 由列定义保证 |
| `ListPlayersByRating` | :many | 按 rating DESC |
| `UpdatePlayerRating` | :exec | (id, rating) |
| `CreateMatch` | :one | 全部快照字段 + played_at + created_at |
| `GetMatchByID` | :one | |
| `ListGlobalMatches` | :many | deleted_at IS NULL，ORDER BY played_at DESC, id DESC，LIMIT ? OFFSET ? |
| `ListPlayerMatches` | :many | (winner_id=? OR loser_id=?) AND deleted_at IS NULL，ORDER BY played_at DESC, id DESC，LIMIT ? OFFSET ? |
| `GetPlayerHistory` | :many | (winner_id=? OR loser_id=?) AND deleted_at IS NULL，ORDER BY played_at ASC, id ASC → (played_at, winner_id, loser_id, winner_rating_after, loser_rating_after) |
| `SoftDeleteMatch` | :exec | (deleted_at, deleted_by, id) WHERE deleted_at IS NULL |
| `RestoreMatch` | :exec | (id) SET deleted_at=NULL, deleted_by=NULL |
| `ListDeletedMatches` | :many | deleted_at IS NOT NULL ORDER BY deleted_at DESC |
| `CountPlayerWinsLosses` | :one | (player_id) → (wins, losses)，仅统计 deleted_at IS NULL |
| `CreateSession` | :exec | (token, player_id, created_at, expires_at) |
| `GetSession` | :one | (token) → session row |
| `DeleteSession` | :exec | (token) |
| `DeleteExpiredSessions` | :exec | (now) |
| `CreateAdminSession` | :exec | (token, created_at, expires_at) |
| `GetAdminSession` | :one | (token) |
| `DeleteAdminSession` | :exec | (token) |
| `GetSetting` | :one | (key) → value |
| `SetSetting` | :exec | (key, value)，UPSERT |

- `db.New(path string) (*sql.DB, error)`：打开 modernc sqlite，设 `PRAGMA foreign_keys=ON, journal_mode=WAL, busy_timeout=5000`；运行 goose Up；返回 `*sql.DB`

## service 层（`go_ultra/internal/service`）

所有 service 构造函数签名：`func NewXxxService(q *sqlc.Queries, db *sql.DB) *XxxService`

### PlayerService
```
LoginOrCreate(ctx, username string) (domain.Player, error)  // trim+校验；已存在则返回，否则按 DefaultRating 创建
GetByUsername(ctx, username string) (domain.Player, error)
GetStats(ctx, playerID int64) (domain.Stats, error)
ListByRating(ctx) ([]domain.Player, error)
```

### MatchService
```
Record(ctx, submitterID int64, opponentUsername string, result string /* "win"|"loss" */, playedAt time.Time) (RecordResult, error)
  // result=win => winner=submitter；用事务 BEGIN IMMEDIATE；读双方 rating；ComputeDelta；插 match；更新双方 rating
  // 校验：submitter != opponent（否则 ErrSelfMatch）
ListGlobal(ctx, limit, offset int) ([]MatchView, error)
ListByPlayer(ctx, playerID int64, limit, offset int) ([]MatchView, error)
History(ctx, playerID int64, createdAt time.Time) ([]HistoryPoint, error)  // prepend (createdAt, DefaultRating)

type RecordResult struct { MatchID int64; WinnerDelta int; LoserDelta int; NewSelfRating int; NewOpponentRating int }
type MatchView struct { ID int64; Opponent string; Result string; RatingBefore int; RatingAfter int; Delta int; PlayedAt time.Time }
type HistoryPoint struct { PlayedAt time.Time; Rating int }
```

### LeaderboardService
```
List(ctx, minGames int) ([]LeaderboardRow, error)
CompareData(ctx, usernames []string) (CompareResult, error)  // series + head_to_head（C(n,2)）

type LeaderboardRow struct { Rank int; Username string; Rating int; Dan int; GamesPlayed int; WinRate float64 }
type CompareSeries struct { Username string; Color string; Points []HistoryPoint }
type HeadToHead struct { A string; B string; AWins int; BWins int }
type CompareResult struct { Series []CompareSeries; HeadToHead []HeadToHead }
```

### AdminService
```
EnsurePassword(ctx) (plaintext string, generated bool, err error)
  // 若 settings 无 admin_password_hash 则生成 16 位随机 + bcrypt 存；返回明文（仅生成时）；已存在返回 ("", false, nil)
VerifyPassword(ctx, pw string) (bool, error)
CreateAdminSession(ctx) (token string, expiresAt time.Time, err error)  // 30 分钟
CheckAdminSession(ctx, token string) (bool, time.Time, error)
SoftDelete(ctx, matchID int64) error  // deleted_by = NULL（管理员非 player）
Restore(ctx, matchID int64) error
```

## session 层（`go_ultra/internal/session`）
```
func NewToken() (string, error)  // crypto/rand 32 字节 → base64.RawURLEncoding
const PlayerSessionTTL = 30 * 24 * time.Hour  // 30 天，滑动续期
const AdminSessionTTL  = 30 * time.Minute
```
Cookie 名：玩家 `go_ultra_session`；管理员 `go_ultra_admin`；HttpOnly, Secure, SameSite=Lax, Path=/

## http 层（`go_ultra/internal/handler` + `middleware`）

- `router.go`：`func NewRouter(deps Deps) *gin.Engine`
  ```go
  type Deps struct {
  	Player      *service.PlayerService
  	Match       *service.MatchService
  	Leaderboard *service.LeaderboardService
  	Admin       *service.AdminService
  	Logger      zerolog.Logger
  }
  ```
- 中间件：`RequestID()`、`Logger(zerolog)`、`Recover()`（panic→500 JSON）、`PlayerAuth()`（注入 playerID 到 ctx）、`AdminAuth()`（校验 admin cookie）
- 统一错误响应：`{ "error": { "code": ..., "message": ... } }`；从 `*domain.Error` 取 Status/Code/Message；非 domain.Error → ErrInternal(500) 并 log Cause
- 路由表严格按 spec §6.1。健康检查：`GET /api/healthz` → 200 `{"status":"ok"}`（审阅采纳，供 start.bat 探测）
- 录入对局：submitter 必为参赛方之一（result 相对 submitter）；opponent 不能等于自己 → ErrSelfMatch
- played_at 校验：允许任意过去时间；拒绝未来时间（> now）→ ErrInvalidParam（审阅采纳）
- compare usernames 上限 10，超出 ErrInvalidParam
- DELETE/restore 幂等返回 204

## 前端（`web/src`）

- `api/types.ts`：TS 接口与后端 DTO 字段一一对应。后端用 `emit_json_tags` → JSON 为 **snake_case**；**前端 types 直接用 snake_case 与 JSON 对齐**（不做转换）
- `api/client.ts`：axios 实例 baseURL `/api`，withCredentials true，响应错误 interceptor 解析 `{error:{code,message}}` → 抛 `ApiError`；401 → 跳 `/login`
- 路由（react-router v6）按 spec §7.1
- 段位前端镜像 `lib/rank.ts`：
  ```
  danOf(rating: number): number   // 与后端 Dan 完全一致：rating<1050→0；(rating-800)//200；>9→9
  danLabel(dan: number): string   // dan===0 ? "未定级" : "段 " + dan
  danColor(dan: number): string   // 段0灰/1-3蓝/4-6紫/7-8金/9红，返回 hex
  ```
- 共享 fixture：`web/src/lib/__fixtures__/rank_cases.csv` 与 `server/internal/domain/testdata/rank_cases.csv` 内容一致（两列 `rating,expected_dan`，含上面所有边界）
- 组件/页面/布局严格按 spec §7.2–§7.5（我的主页=大曲线+右侧栏；排行榜=表格+顶部领奖台；对比=左侧栏控制+右大曲线+下方头对头卡片；录入=标准表单+Elo 预览）
- ECharts 暗色；5 色板 hex：`["#4a9eff", "#7fd6a3", "#8b5cf6", "#e0c47d", "#f08080"]`
- `lib/elo-preview.ts`：与后端 `ComputeDelta` 同公式，供录入弹窗实时预览，必须严格 TDD

## 任务撰写规则（所有 agent 通用）

硬性要求（违反即失败）：

1. 全程中文叙述，代码与命令用英文。
2. 每个 Task 用如下结构，步骤用 markdown 复选框 `- [ ]`：
   ```
   ### Task N: <名称>
   **Files:**
   - Create: <精确路径>
   - Test: <精确路径>
   ```
   每个 Task 内部步骤严格遵循 TDD：写失败测试 → 运行确认失败（给出命令与预期失败信息）→ 写最小实现 → 运行确认通过 → commit。
3. **禁止任何占位符**：不准出现 "TODO"/"略"/"类似上文"/"自行补充"/"添加适当的错误处理" 等。凡涉及代码的步骤，必须给出**完整可粘贴的代码**（不是省略号片段）。命令步骤给出确切命令和预期输出。
4. 严格使用本契约的函数名/类型/签名/查询名，不得自创或改名。
5. 每个 Task 结尾必须有一个 commit 步骤，给出确切的 `git add` + `git commit -m "..."`（conventional commits 风格）。
6. 路径全部以仓库根 `go_ultra/` 为基准的相对路径。
7. 测试框架：后端 Go 标准 `testing`（table-driven 优先）；前端 Vitest + `@testing-library/react`。
8. 只输出你负责部分的 markdown 正文（从 `## 阶段X` 标题开始），不要输出整个文件头，不要寒暄，不要解释你在做什么。
