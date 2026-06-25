# go_ultra 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建通用 1v1 竞技游戏等级分系统（go_ultra）的 MVP —— Go+Gin+SQLite 后端、React+TS+ECharts 前端、本机 Caddy + Cloudflare Tunnel 部署。

**Architecture:** 前后端分离。后端按 domain（纯业务/Elo/段位）→ db（sqlc + goose 迁移）→ service（事务编排）→ handler（Gin + 中间件）分层；前端按 api/lib/components/pages 组织，React Query 管数据。Caddy 反代 `/api` 到 :8080、服务前端静态文件，Cloudflare Tunnel 暴露公网。

**Tech Stack:** Go 1.22 / Gin / modernc SQLite / sqlc / goose / zerolog / bcrypt；React 18 / TypeScript / Vite / Tailwind / shadcn/ui / ECharts / TanStack Query / axios / react-hook-form / zod；Caddy / cloudflared。

**设计规范来源：** `docs/superpowers/specs/2026-06-25-go-ultra-design.md`
**接口契约（实现期唯一真相源）：** `docs/superpowers/plans/_contract.md` —— 实现任何阶段前先通读，所有类型/函数/查询名以契约为准。

---

## 阶段与依赖关系

| 阶段 | 内容 | 依赖 |
|---|---|---|
| 0 | 项目脚手架与工具链 | — |
| 1 | domain 层（类型 / Elo / 段位 / 错误） | 0 |
| 2 | db 层（迁移 / sqlc / 连接） | 0 |
| 3 | service 层（玩家 / 对局 / 排行榜 / 管理员） | 1, 2 |
| 4 | http 层（session / 中间件 / handler / 装配 main.go） | 3 |
| 5 | 前端基础（脚手架 / api / lib / 路由） | 4（联调） |
| 6 | 前端页面与组件 | 5 |
| 7 | 部署与运维脚本（含 reset-admin-password 子命令） | 4, 6 |

每个阶段内部任务严格 TDD：写失败测试 → 确认失败 → 最小实现 → 确认通过 → commit。

### 合并约定（消解跨段冲突，实现者务必遵守）

并行起草各阶段时发现以下边界，已在此统一裁定：

1. **`server/cmd/go_ultra/main.go` 的所有权**：由**阶段 4** 创建并拥有（装配 db+service+router、`EnsurePassword(context.Background())`、监听 :8080、DB 路径来源 `defaultDBPath` + `GO_ULTRA_DB` 环境变量）。**阶段 7 不重写 main.go**，只**增量**加入子命令分发（`reset-admin-password`），复用阶段 4 的 `buildRouter`/装配逻辑，禁止自造 `startHTTP`/`runServe`/`select{}` 占位或第二套 `dbPath` 常量。
2. **AdminService setting key**：在 service 层（阶段 3）定义常量 `const adminPasswordHashKey = "admin_password_hash"`，`EnsurePassword`/`VerifyPassword` 及阶段 7 的 `reset-admin-password` 全部引用此常量，禁止散落字符串字面量。
3. **共享 fixture `rank_cases.csv`**：后端文件 `server/internal/domain/testdata/rank_cases.csv` 由**阶段 1** 创建并提交；**阶段 5** 只创建前端 `web/src/lib/__fixtures__/rank_cases.csv` 并与后端文件 `diff` 校验一致，不重复创建/提交后端文件。
4. **commit scope 统一**：涉及 `cmd/go_ultra` 的提交一律用 scope `cmd`（不用 `cli`）。
5. **History DTO**：service 层 `HistoryPoint` 的 JSON tag 必须恰为 `played_at` / `rating`，与前端 `api/types.ts` 的 `HistoryPoint` 对齐。

---

## 阶段 0: 项目脚手架与工具链

> 本阶段目标：让 `server/` 成为一个能编译、能 `go test ./...`、依赖齐全的 Go 工程。本阶段不写任何业务逻辑，只搭骨架。完成后，后续阶段每个 Task 都能在已就绪的依赖与目录上直接落地。

### Task 0.1: 初始化 git 仓库与 server/go.mod

**Files:**
- Create: `server/go.mod`
- Create: `.gitignore`

步骤（本 Task 是纯脚手架，无可执行测试代码；"测试"即 `go mod verify` 与 `go build`，在 Task 0.3 完成后跑通）：

- [ ] 确认仓库根 `go_ultra/` 下还没有 git 仓库；若没有，在仓库根执行初始化：
  ```bash
  git init
  ```
  预期输出包含 `Initialized empty Git repository in .../go_ultra/.git/`。

- [ ] 在仓库根创建 `.gitignore`，完整内容如下（覆盖 Go 产物、SQLite 文件、日志、IDE、前端产物）：
  ```gitignore
  # Go
  /server/go_ultra
  /server/go_ultra.exe
  *.test
  *.out

  # SQLite
  *.db
  *.db-wal
  *.db-shm

  # logs
  /logs/

  # frontend
  /web/node_modules/
  /web/dist/

  # IDE / OS
  .idea/
  .vscode/
  .DS_Store
  Thumbs.db
  ```

- [ ] 创建 `server/go.mod`，完整内容如下（module 名严格为 `go_ultra`，Go 版本 1.22）。注意：require 块中的版本号会在 Task 0.2 执行 `go get` 后由 go 工具自动校正/补全 `go.sum`，这里先写出确定的主版本，后续 `go mod tidy` 会落定精确补丁号：
  ```
  module go_ultra

  go 1.22

  require (
  	github.com/gin-gonic/gin v1.10.0
  	github.com/go-playground/validator/v10 v10.22.1
  	github.com/google/uuid v1.6.0
  	github.com/pressly/goose/v3 v3.22.1
  	github.com/rs/zerolog v1.33.0
  	golang.org/x/crypto v0.31.0
  	modernc.org/sqlite v1.34.1
  )
  ```
  说明：`gin` 用于 http 层（阶段 3+），`validator` 用于校验，`uuid` 供 request_id 中间件，`goose/v3` 供迁移，`zerolog` 供日志，`x/crypto` 供 bcrypt，`modernc.org/sqlite` 是纯 Go 的 SQLite 驱动（无 CGO，Windows 下零工具链）。

- [ ] 提交骨架第一步：
  ```bash
  git add .gitignore server/go.mod
  git commit -m "chore: init repo and server go.mod (module go_ultra)"
  ```

### Task 0.2: 拉取全部 Go 依赖

**Files:**
- Modify: `server/go.mod`
- Create: `server/go.sum`

步骤（所有 `go` 命令都在 `server/` 目录下执行；本指南所有命令均假设当前工作目录为 `server/`，除非特别说明）：

- [ ] 在 `server/` 目录执行以下 `go get` 命令逐个拉取依赖（modernc.org/sqlite 是纯 Go 驱动，无需 CGO，因此 `CGO_ENABLED` 无所谓）：
  ```bash
  go get github.com/gin-gonic/gin@v1.10.0
  go get github.com/go-playground/validator/v10@v10.22.1
  go get github.com/google/uuid@v1.6.0
  go get github.com/pressly/goose/v3@v3.22.1
  go get github.com/rs/zerolog@v1.33.0
  go get golang.org/x/crypto@v0.31.0
  go get modernc.org/sqlite@v1.34.1
  ```
  每条预期输出形如 `go: added github.com/... vX.Y.Z`（或 `go: upgraded ...`）。

- [ ] 整理依赖、生成/补全 `go.sum` 并下载校验：
  ```bash
  go mod tidy
  go mod verify
  ```
  `go mod tidy` 预期无报错并补全间接依赖；`go mod verify` 预期输出 `all modules verified`。

- [ ] 安装 sqlc 与 goose 命令行工具到本机（供后续阶段生成代码与手动跑迁移；二者作为 CLI 工具安装，不进 go.mod 的依赖图）：
  ```bash
  go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
  go install github.com/pressly/goose/v3/cmd/goose@v3.22.1
  ```
  安装后确认可执行（`$GOPATH/bin` 或 `$HOME/go/bin` 需在 PATH 中）：
  ```bash
  sqlc version
  goose --version
  ```
  预期分别输出 `v1.27.0` 与 `goose version: v3.22.1`（或等价版本字符串）。

- [ ] 提交依赖：
  ```bash
  git add server/go.mod server/go.sum
  git commit -m "chore: add go dependencies (gin, sqlc runtime, goose, zerolog, validator, bcrypt, modernc sqlite)"
  ```

### Task 0.3: 建立 internal 目录骨架与最小可编译 main

**Files:**
- Create: `server/internal/domain/.gitkeep`
- Create: `server/internal/config/.gitkeep`
- Create: `server/internal/db/migrations/.gitkeep`
- Create: `server/internal/service/.gitkeep`
- Create: `server/internal/handler/.gitkeep`
- Create: `server/internal/middleware/.gitkeep`
- Create: `server/internal/session/.gitkeep`
- Create: `server/queries/.gitkeep`
- Create: `server/cmd/go_ultra/main.go`
- Test: 编译验证（`go build`），无独立 `_test.go`

> 说明：`internal/db/sqlc/` 目录由 sqlc 在阶段 2 生成，这里不预建。各空目录放 `.gitkeep` 仅为让 git 跟踪空目录；当目录内出现真实 `.go` 文件后可删除对应 `.gitkeep`。

步骤：

- [ ] 创建以下空占位文件，使 git 跟踪目录结构（每个文件内容为空）：
  - `server/internal/domain/.gitkeep`
  - `server/internal/config/.gitkeep`
  - `server/internal/db/migrations/.gitkeep`
  - `server/internal/service/.gitkeep`
  - `server/internal/handler/.gitkeep`
  - `server/internal/middleware/.gitkeep`
  - `server/internal/session/.gitkeep`
  - `server/queries/.gitkeep`

- [ ] 创建 `server/cmd/go_ultra/main.go`，完整内容如下（本阶段仅打印版本号；后续阶段在此装配 db、service、router）：
  ```go
  package main

  import "fmt"

  // Version 是构建版本号，后续可由 -ldflags 注入覆盖。
  var Version = "0.1.0-dev"

  func main() {
  	fmt.Printf("go_ultra %s\n", Version)
  }
  ```

- [ ] 运行编译验证（在 `server/` 目录）。先确认当前可编译：
  ```bash
  go build ./...
  ```
  预期：无任何输出、退出码 0（编译通过）。

- [ ] 运行二进制确认打印版本：
  ```bash
  go run ./cmd/go_ultra
  ```
  预期标准输出：
  ```
  go_ultra 0.1.0-dev
  ```

- [ ] 运行测试入口（此时无测试，仅确认 `go test` 能扫描全树而不报错）：
  ```bash
  go test ./...
  ```
  预期：每个包输出 `no test files`（无 FAIL）。

- [ ] 提交骨架：
  ```bash
  git add server/internal server/queries server/cmd/go_ultra/main.go
  git commit -m "chore: scaffold internal package tree and minimal main entrypoint"
  ```

---

## 阶段 1: domain 层（纯业务模型 + Elo + 段位 + 错误）

> 本阶段实现 `internal/domain` 包：纯函数与值类型，无任何外部依赖（不 import db / gin）。这是"换场景只改一处"约束的落点 —— K 因子、起始分、段位边界、段位标签全部集中在此。目标覆盖率 100%（行 + 分支）。
>
> 每个 Task 严格 TDD：先写 `_test.go` 跑出失败 → 写最小实现 → 跑通 → commit。所有 `go test` 命令在 `server/` 目录执行。

### Task 1.1: domain 类型定义（types.go）

**Files:**
- Create: `server/internal/domain/types.go`
- Test: `server/internal/domain/types_test.go`

> `types.go` 是纯数据结构（无方法），没有逻辑分支可"断言行为"。为满足"先有失败测试"的 TDD 节奏，这里写一个**编译期 + 字段存在性**测试：测试构造各结构体并读取字段，若结构体未定义则编译失败（即测试失败）。

步骤：

- [ ] 先写失败测试 `server/internal/domain/types_test.go`，完整内容：
  ```go
  package domain

  import (
  	"testing"
  	"time"
  )

  func TestPlayerFields(t *testing.T) {
  	now := time.Now().UTC()
  	p := Player{ID: 1, Username: "alice", Rating: 1500, CreatedAt: now}
  	if p.ID != 1 || p.Username != "alice" || p.Rating != 1500 || !p.CreatedAt.Equal(now) {
  		t.Fatalf("Player fields not assignable as expected: %+v", p)
  	}
  }

  func TestStatsFields(t *testing.T) {
  	s := Stats{Wins: 3, Losses: 2, WinRate: 0.6, CurrentStreak: 1, LongestStreak: 4}
  	if s.Wins != 3 || s.Losses != 2 || s.WinRate != 0.6 || s.CurrentStreak != 1 || s.LongestStreak != 4 {
  		t.Fatalf("Stats fields not assignable as expected: %+v", s)
  	}
  }

  func TestMatchFields(t *testing.T) {
  	now := time.Now().UTC()
  	delBy := int64(7)
  	m := Match{
  		ID:                 10,
  		WinnerID:           1,
  		LoserID:            2,
  		SubmitterID:        1,
  		WinnerRatingBefore: 1500,
  		LoserRatingBefore:  1500,
  		WinnerRatingAfter:  1508,
  		LoserRatingAfter:   1492,
  		WinnerDelta:        8,
  		LoserDelta:         -8,
  		PlayedAt:           now,
  		CreatedAt:          now,
  		DeletedAt:          &now,
  		DeletedBy:          &delBy,
  	}
  	if m.WinnerDelta+m.LoserDelta != 0 {
  		t.Fatalf("expected zero-sum deltas, got %d and %d", m.WinnerDelta, m.LoserDelta)
  	}
  	if m.DeletedAt == nil || *m.DeletedBy != 7 {
  		t.Fatalf("pointer fields not assignable as expected: %+v", m)
  	}
  }
  ```

- [ ] 运行确认失败（`types.go` 尚不存在，编译失败）：
  ```bash
  go test ./internal/domain/
  ```
  预期失败信息形如：`undefined: Player`（以及 `undefined: Stats`、`undefined: Match`），`FAIL go_ultra/internal/domain [build failed]`。

- [ ] 写最小实现 `server/internal/domain/types.go`，完整内容（严格照契约）：
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

- [ ] 运行确认通过：
  ```bash
  go test ./internal/domain/
  ```
  预期：`ok  	go_ultra/internal/domain`。

- [ ] 提交：
  ```bash
  git add server/internal/domain/types.go server/internal/domain/types_test.go
  git commit -m "feat(domain): add core value types (Player, Stats, Match)"
  ```

### Task 1.2: Elo 算法（elo.go）

**Files:**
- Create: `server/internal/domain/elo.go`
- Test: `server/internal/domain/elo_test.go`

> 覆盖契约要求：分差 0 / +400 / −400 / 极端值；`ComputeDelta` 用 `math.Round`（half away from zero）；loser delta = `-ComputeDelta(...)` 的零和性。

步骤：

- [ ] 先写失败测试 `server/internal/domain/elo_test.go`，完整内容。注意 `ExpectedScore` 是浮点，用容差比较；`ComputeDelta` 在 K=16 时：分差 0 → `round(16*0.5)=8`；A 比 B 高 400（即 winner=A 高分）→ E_winner≈0.909 → `round(16*0.0909)=round(1.4545)=1`；A 比 B 低 400（winner 是低分方）→ E_winner≈0.0909 → `round(16*0.909)=round(14.545)=15`：
  ```go
  package domain

  import (
  	"math"
  	"testing"
  )

  func TestExpectedScore(t *testing.T) {
  	const eps = 1e-9
  	tests := []struct {
  		name     string
  		ratingA  int
  		ratingB  int
  		expected float64
  	}{
  		{"equal ratings -> 0.5", 1500, 1500, 0.5},
  		{"A higher by 400 -> ~0.909", 1900, 1500, 1.0 / (1.0 + math.Pow(10, -400.0/400.0))},
  		{"A lower by 400 -> ~0.091", 1500, 1900, 1.0 / (1.0 + math.Pow(10, 400.0/400.0))},
  		{"extreme A dominates", 5000, 0, 1.0 / (1.0 + math.Pow(10, -5000.0/400.0))},
  		{"extreme A crushed", 0, 5000, 1.0 / (1.0 + math.Pow(10, 5000.0/400.0))},
  	}
  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			got := ExpectedScore(tt.ratingA, tt.ratingB)
  			if math.Abs(got-tt.expected) > eps {
  				t.Fatalf("ExpectedScore(%d,%d) = %v, want %v", tt.ratingA, tt.ratingB, got, tt.expected)
  			}
  		})
  	}
  }

  func TestExpectedScoreSymmetry(t *testing.T) {
  	// E_A + E_B == 1
  	const eps = 1e-9
  	pairs := [][2]int{{1500, 1500}, {1900, 1500}, {1500, 1900}, {2400, 1000}, {0, 5000}}
  	for _, p := range pairs {
  		ea := ExpectedScore(p[0], p[1])
  		eb := ExpectedScore(p[1], p[0])
  		if math.Abs(ea+eb-1.0) > eps {
  			t.Fatalf("E_A+E_B != 1 for %v: %v + %v", p, ea, eb)
  		}
  	}
  }

  func TestComputeDelta(t *testing.T) {
  	tests := []struct {
  		name          string
  		winnerRating  int
  		loserRating   int
  		expectedDelta int
  	}{
  		{"equal ratings", 1500, 1500, 8},                 // round(16*0.5)=8
  		{"winner higher by 400", 1900, 1500, 1},          // round(16*0.0909..)=round(1.4545)=1
  		{"winner lower by 400 (upset)", 1500, 1900, 15},  // round(16*0.9090..)=round(14.545)=15
  		{"extreme upset (winner crushed favorite)", 0, 5000, 16}, // round(16*~1.0)=16
  		{"extreme expected win", 5000, 0, 0},             // round(16*~0.0)=0
  	}
  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			got := ComputeDelta(tt.winnerRating, tt.loserRating)
  			if got != tt.expectedDelta {
  				t.Fatalf("ComputeDelta(%d,%d) = %d, want %d", tt.winnerRating, tt.loserRating, got, tt.expectedDelta)
  			}
  		})
  	}
  }

  func TestComputeDeltaZeroSum(t *testing.T) {
  	// winner delta + loser delta（= -ComputeDelta）必须等于 0
  	pairs := [][2]int{{1500, 1500}, {1900, 1500}, {1500, 1900}, {2400, 1000}, {0, 5000}}
  	for _, p := range pairs {
  		d := ComputeDelta(p[0], p[1])
  		loserDelta := -d
  		if d+loserDelta != 0 {
  			t.Fatalf("zero-sum violated for %v: winner=%d loser=%d", p, d, loserDelta)
  		}
  	}
  }

  func TestEloConstants(t *testing.T) {
  	if DefaultRating != 1500 {
  		t.Fatalf("DefaultRating = %d, want 1500", DefaultRating)
  	}
  	if KFactor != 16 {
  		t.Fatalf("KFactor = %d, want 16", KFactor)
  	}
  }
  ```

- [ ] 运行确认失败：
  ```bash
  go test ./internal/domain/
  ```
  预期失败信息形如：`undefined: ExpectedScore`、`undefined: ComputeDelta`、`undefined: DefaultRating`、`undefined: KFactor`，`FAIL ... [build failed]`。

- [ ] 写最小实现 `server/internal/domain/elo.go`，完整内容（所有可变常量集中于此，换场景只改这里）：
  ```go
  package domain

  import "math"

  // DefaultRating 是新玩家的起始等级分。换场景时改此常量。
  const DefaultRating = 1500

  // KFactor 控制每局分数变动幅度。换场景时改此常量。
  const KFactor = 16

  // ExpectedScore 返回 A 对 B 的期望胜率：1/(1+10^((B-A)/400))。
  func ExpectedScore(ratingA, ratingB int) float64 {
  	return 1.0 / (1.0 + math.Pow(10, float64(ratingB-ratingA)/400.0))
  }

  // ComputeDelta 返回胜者应获得的分数变动（正整数或 0），
  // 采用 math.Round（half away from zero）。败者变动为其相反数。
  func ComputeDelta(winnerRating, loserRating int) int {
  	eWinner := ExpectedScore(winnerRating, loserRating)
  	return int(math.Round(KFactor * (1.0 - eWinner)))
  }
  ```

- [ ] 运行确认通过：
  ```bash
  go test ./internal/domain/
  ```
  预期：`ok  	go_ultra/internal/domain`。

- [ ] 提交：
  ```bash
  git add server/internal/domain/elo.go server/internal/domain/elo_test.go
  git commit -m "feat(domain): implement Elo ExpectedScore and ComputeDelta with zero-sum guarantee"
  ```

### Task 1.3: 段位映射（rank.go）

**Files:**
- Create: `server/internal/domain/rank.go`
- Test: `server/internal/domain/rank_test.go`

> 必须 table-driven 覆盖契约列出的**全部边界**：1049→0, 1050→1, 1199→1, 1200→2, 1399→2, 1400→3, 1500→3, 1599→3, 1600→4, 2399→7, 2400→8, 2599→8, 2600→9, 5000→9。

步骤：

- [ ] 先写失败测试 `server/internal/domain/rank_test.go`，完整内容：
  ```go
  package domain

  import "testing"

  func TestDan(t *testing.T) {
  	tests := []struct {
  		rating   int
  		expected int
  	}{
  		{1049, 0},
  		{1050, 1},
  		{1199, 1},
  		{1200, 2},
  		{1399, 2},
  		{1400, 3},
  		{1500, 3},
  		{1599, 3},
  		{1600, 4},
  		{2399, 7},
  		{2400, 8},
  		{2599, 8},
  		{2600, 9},
  		{5000, 9},
  	}
  	for _, tt := range tests {
  		t.Run(itoa(tt.rating), func(t *testing.T) {
  			got := Dan(tt.rating)
  			if got != tt.expected {
  				t.Fatalf("Dan(%d) = %d, want %d", tt.rating, got, tt.expected)
  			}
  		})
  	}
  }

  func TestDanBelowFloorBoundary(t *testing.T) {
  	// RankFloor 本身应映射为段 1，低一分应为段 0
  	if Dan(RankFloor) != 1 {
  		t.Fatalf("Dan(RankFloor=%d) = %d, want 1", RankFloor, Dan(RankFloor))
  	}
  	if Dan(RankFloor-1) != 0 {
  		t.Fatalf("Dan(RankFloor-1=%d) = %d, want 0", RankFloor-1, Dan(RankFloor-1))
  	}
  }

  func TestRankFloorConstant(t *testing.T) {
  	if RankFloor != 1050 {
  		t.Fatalf("RankFloor = %d, want 1050", RankFloor)
  	}
  }

  // itoa 是测试内部小工具，避免引入 strconv 仅为子测试命名。
  func itoa(n int) string {
  	if n == 0 {
  		return "0"
  	}
  	neg := n < 0
  	if neg {
  		n = -n
  	}
  	var buf [20]byte
  	i := len(buf)
  	for n > 0 {
  		i--
  		buf[i] = byte('0' + n%10)
  		n /= 10
  	}
  	if neg {
  		i--
  		buf[i] = '-'
  	}
  	return string(buf[i:])
  }
  ```

- [ ] 运行确认失败：
  ```bash
  go test ./internal/domain/
  ```
  预期失败信息形如：`undefined: Dan`、`undefined: RankFloor`，`FAIL ... [build failed]`。

- [ ] 写最小实现 `server/internal/domain/rank.go`，完整内容（段位边界集中于此，换场景只改这里）：
  ```go
  package domain

  // RankFloor 是显示段位的最低分；低于此值返回段 0（UI 不显示徽章）。
  // 换场景时连同 Dan 的边界逻辑一起调整。
  const RankFloor = 1050

  // Dan 把等级分映射为段位：
  //   rating < RankFloor        -> 0（未定级）
  //   tier = (rating-800)/200，封顶 9
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

- [ ] 运行确认通过：
  ```bash
  go test ./internal/domain/
  ```
  预期：`ok  	go_ultra/internal/domain`。

- [ ] 提交：
  ```bash
  git add server/internal/domain/rank.go server/internal/domain/rank_test.go
  git commit -m "feat(domain): implement Dan rank mapping with full boundary coverage"
  ```

### Task 1.4: 领域错误类型（errors.go）

**Files:**
- Create: `server/internal/domain/errors.go`
- Test: `server/internal/domain/errors_test.go`

> 覆盖 `Error()` 字符串、`WithCause`（返回副本且原值不被改动）、所有预定义错误的 Code/Status 一致性。

步骤：

- [ ] 先写失败测试 `server/internal/domain/errors_test.go`，完整内容：
  ```go
  package domain

  import (
  	"errors"
  	"strings"
  	"testing"
  )

  func TestErrorString(t *testing.T) {
  	e := &Error{Code: "PLAYER_NOT_FOUND", Message: "玩家不存在", Status: 404}
  	got := e.Error()
  	if !strings.Contains(got, "PLAYER_NOT_FOUND") || !strings.Contains(got, "玩家不存在") {
  		t.Fatalf("Error() = %q, want it to contain code and message", got)
  	}
  }

  func TestWithCauseReturnsCopy(t *testing.T) {
  	cause := errors.New("disk exploded")
  	withCause := ErrInternal.WithCause(cause)

  	// 副本带上 Cause
  	if withCause.Cause != cause {
  		t.Fatalf("WithCause did not attach cause; got %v", withCause.Cause)
  	}
  	// 原始预定义值不被污染
  	if ErrInternal.Cause != nil {
  		t.Fatalf("WithCause mutated the original sentinel; ErrInternal.Cause = %v", ErrInternal.Cause)
  	}
  	// 副本保留原 Code/Message/Status
  	if withCause.Code != ErrInternal.Code || withCause.Message != ErrInternal.Message || withCause.Status != ErrInternal.Status {
  		t.Fatalf("WithCause changed code/message/status: %+v", withCause)
  	}
  	// 是不同的指针
  	if withCause == ErrInternal {
  		t.Fatalf("WithCause returned the same pointer, expected a copy")
  	}
  }

  func TestWithCauseErrorStringStillWorks(t *testing.T) {
  	e := ErrPlayerNotFound.WithCause(errors.New("row not found"))
  	if !strings.Contains(e.Error(), "PLAYER_NOT_FOUND") {
  		t.Fatalf("Error() after WithCause = %q", e.Error())
  	}
  }

  func TestPredefinedErrors(t *testing.T) {
  	tests := []struct {
  		err    *Error
  		code   string
  		status int
  	}{
  		{ErrPlayerNotFound, "PLAYER_NOT_FOUND", 404},
  		{ErrMatchNotFound, "MATCH_NOT_FOUND", 404},
  		{ErrSelfMatch, "SELF_MATCH", 409},
  		{ErrNotAuthenticated, "NOT_AUTHENTICATED", 401},
  		{ErrAdminRequired, "ADMIN_REQUIRED", 403},
  		{ErrInvalidBody, "INVALID_BODY", 400},
  		{ErrInvalidParam, "INVALID_PARAM", 400},
  		{ErrInternal, "INTERNAL", 500},
  	}
  	for _, tt := range tests {
  		t.Run(tt.code, func(t *testing.T) {
  			if tt.err == nil {
  				t.Fatalf("predefined error %s is nil", tt.code)
  			}
  			if tt.err.Code != tt.code {
  				t.Fatalf("Code = %q, want %q", tt.err.Code, tt.code)
  			}
  			if tt.err.Status != tt.status {
  				t.Fatalf("Status = %d, want %d (code %s)", tt.err.Status, tt.status, tt.code)
  			}
  			if tt.err.Message == "" {
  				t.Fatalf("Message empty for code %s", tt.code)
  			}
  		})
  	}
  }
  ```

- [ ] 运行确认失败：
  ```bash
  go test ./internal/domain/
  ```
  预期失败信息形如：`undefined: Error`、`undefined: ErrInternal`、`undefined: ErrPlayerNotFound` 等，`FAIL ... [build failed]`。

- [ ] 写最小实现 `server/internal/domain/errors.go`，完整内容（严格照契约的 Code/Message/Status）：
  ```go
  package domain

  // Error 是统一的领域错误类型：service 层只抛它，handler 层据此映射 HTTP 响应。
  type Error struct {
  	Code    string
  	Message string
  	Status  int
  	Cause   error
  }

  // Error 实现 error 接口，输出便于日志排查的字符串。
  func (e *Error) Error() string {
  	if e.Cause != nil {
  		return e.Code + ": " + e.Message + ": " + e.Cause.Error()
  	}
  	return e.Code + ": " + e.Message
  }

  // WithCause 返回一个带上底层原因的副本，不修改原始（预定义 sentinel）值。
  func (e *Error) WithCause(err error) *Error {
  	cp := *e
  	cp.Cause = err
  	return &cp
  }

  var (
  	ErrPlayerNotFound   = &Error{Code: "PLAYER_NOT_FOUND", Message: "玩家不存在", Status: 404}
  	ErrMatchNotFound    = &Error{Code: "MATCH_NOT_FOUND", Message: "对局不存在", Status: 404}
  	ErrSelfMatch        = &Error{Code: "SELF_MATCH", Message: "不能和自己对局", Status: 409}
  	ErrNotAuthenticated = &Error{Code: "NOT_AUTHENTICATED", Message: "未登录", Status: 401}
  	ErrAdminRequired    = &Error{Code: "ADMIN_REQUIRED", Message: "需要管理员权限", Status: 403}
  	ErrInvalidBody      = &Error{Code: "INVALID_BODY", Message: "请求体无效", Status: 400}
  	ErrInvalidParam     = &Error{Code: "INVALID_PARAM", Message: "参数无效", Status: 400}
  	ErrInternal         = &Error{Code: "INTERNAL", Message: "服务器内部错误", Status: 500}
  )
  ```

- [ ] 运行确认通过：
  ```bash
  go test ./internal/domain/
  ```
  预期：`ok  	go_ultra/internal/domain`。

- [ ] 提交：
  ```bash
  git add server/internal/domain/errors.go server/internal/domain/errors_test.go
  git commit -m "feat(domain): add Error type, WithCause, and predefined sentinel errors"
  ```

### Task 1.5: 段位 CSV fixture 与 CSV 驱动测试

**Files:**
- Create: `server/internal/domain/testdata/rank_cases.csv`
- Test: `server/internal/domain/rank_csv_test.go`

> 该 CSV 是后端与前端 `lib/rank.ts` 共享的"唯一段位边界真相源"（契约要求两份内容一致）。本 Task 写一个**从 CSV 读取驱动** `Dan` 的测试，保证 fixture 与实现永远对齐。CSV 两列：`rating,expected_dan`，含全部契约边界。

步骤：

- [ ] 创建 fixture `server/internal/domain/testdata/rank_cases.csv`，完整内容（首行是表头，随后逐行覆盖全部契约边界；前端共享文件 `web/src/lib/__fixtures__/rank_cases.csv` 必须与此**逐字节一致**，由前端阶段负责复制）：
  ```csv
  rating,expected_dan
  1049,0
  1050,1
  1199,1
  1200,2
  1399,2
  1400,3
  1500,3
  1599,3
  1600,4
  2399,7
  2400,8
  2599,8
  2600,9
  5000,9
  ```

- [ ] 先写失败测试 `server/internal/domain/rank_csv_test.go`，完整内容（用标准库 `encoding/csv` 读取，跳过表头，逐行断言 `Dan(rating)==expected`）：
  ```go
  package domain

  import (
  	"encoding/csv"
  	"os"
  	"strconv"
  	"testing"
  )

  func TestDanFromCSVFixture(t *testing.T) {
  	const path = "testdata/rank_cases.csv"
  	f, err := os.Open(path)
  	if err != nil {
  		t.Fatalf("open fixture %s: %v", path, err)
  	}
  	defer f.Close()

  	rows, err := csv.NewReader(f).ReadAll()
  	if err != nil {
  		t.Fatalf("parse csv %s: %v", path, err)
  	}
  	if len(rows) < 2 {
  		t.Fatalf("fixture %s has no data rows", path)
  	}

  	header := rows[0]
  	if len(header) != 2 || header[0] != "rating" || header[1] != "expected_dan" {
  		t.Fatalf("unexpected header %v, want [rating expected_dan]", header)
  	}

  	for i, row := range rows[1:] {
  		lineNo := i + 2 // 1-based, accounting for header
  		if len(row) != 2 {
  			t.Fatalf("line %d: expected 2 columns, got %d (%v)", lineNo, len(row), row)
  		}
  		rating, err := strconv.Atoi(row[0])
  		if err != nil {
  			t.Fatalf("line %d: bad rating %q: %v", lineNo, row[0], err)
  		}
  		expected, err := strconv.Atoi(row[1])
  		if err != nil {
  			t.Fatalf("line %d: bad expected_dan %q: %v", lineNo, row[1], err)
  		}
  		if got := Dan(rating); got != expected {
  			t.Fatalf("line %d: Dan(%d) = %d, want %d", lineNo, rating, got, expected)
  		}
  	}
  }
  ```

- [ ] 运行确认失败 —— 这里有两种可能的"先失败"情形，任一出现即说明测试在起作用：
  - 若先创建测试但**尚未**创建 CSV，则失败于 `open fixture testdata/rank_cases.csv: ... no such file or directory`；
  - 若按本 Task 顺序 CSV 已存在但 `Dan` 实现还未就绪（在隔离回放本 Task 时），失败于 `undefined: Dan`。
  按本计划顺序（Task 1.3 已实现 `Dan`、本 Task 已先建 CSV），为制造一次明确失败，临时把 CSV 中 `1500,3` 改成 `1500,4` 后运行：
  ```bash
  go test ./internal/domain/ -run TestDanFromCSVFixture
  ```
  预期失败信息形如：`line 8: Dan(1500) = 3, want 4`，`FAIL go_ultra/internal/domain`。随后把该行改回 `1500,3`。

- [ ] 运行确认通过（恢复 CSV 后）：
  ```bash
  go test ./internal/domain/ -run TestDanFromCSVFixture
  ```
  预期：`ok  	go_ultra/internal/domain`。

- [ ] 跑整个 domain 包并查看覆盖率，确认 domain 层达到 100% 目标：
  ```bash
  go test -cover ./internal/domain/
  ```
  预期：`ok  	go_ultra/internal/domain  ...  coverage: 100.0% of statements`。

- [ ] 提交：
  ```bash
  git add server/internal/domain/testdata/rank_cases.csv server/internal/domain/rank_csv_test.go
  git commit -m "test(domain): add shared rank_cases.csv fixture and CSV-driven Dan test"
  ```

---

## 阶段 2: db 层（迁移 + sqlc 查询 + 连接装配）

> 本阶段产出：goose 迁移、sqlc 配置与查询、`db.New` 连接函数（含 PRAGMA + 自动迁移）及其集成测试。SQLite 驱动用纯 Go 的 `modernc.org/sqlite`（驱动名 `sqlite`）。所有时间列为 `TEXT`（存 RFC3339）。
>
> 注意 sqlc 的代码生成不是 TDD 单元，而是一个确定性的"输入 SQL → 生成 Go"过程；本阶段的 TDD 落在 `db.New` 的集成测试（用临时文件库跑迁移并断言表存在）。`go test` 命令在 `server/` 目录执行。

### Task 2.1: goose 迁移文件（00001_init.sql）

**Files:**
- Create: `server/internal/db/migrations/00001_init.sql`
- Test: 通过 Task 2.5 的 `db.New` 集成测试验证（本 Task 内先用 `goose` CLI 手动验证语法）

> 表结构严格按 spec §4.1，但所有时间列改 `TEXT`；matches 增加契约要求的 3 个 CHECK；额外索引 `idx_sessions_expires`。

步骤：

- [ ] 创建 `server/internal/db/migrations/00001_init.sql`，完整内容如下（包含 `-- +goose Up` 与 `-- +goose Down`；每条建表/索引语句用 `-- +goose StatementBegin/End` 包裹以兼容 goose 解析）：
  ```sql
  -- +goose Up
  -- +goose StatementBegin
  CREATE TABLE players (
      id              INTEGER PRIMARY KEY AUTOINCREMENT,
      username        TEXT NOT NULL UNIQUE COLLATE NOCASE,
      rating          INTEGER NOT NULL DEFAULT 1500,
      created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
  );
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_players_rating ON players(rating DESC);
  -- +goose StatementEnd

  -- +goose StatementBegin
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

      played_at       TEXT NOT NULL,
      created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),

      deleted_at      TEXT,
      deleted_by      INTEGER REFERENCES players(id),

      CHECK (winner_id != loser_id),
      CHECK (winner_rating_after = winner_rating_before + winner_delta),
      CHECK (loser_rating_after  = loser_rating_before  + loser_delta),
      CHECK (winner_delta + loser_delta = 0)
  );
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_matches_winner ON matches(winner_id, played_at DESC);
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_matches_loser ON matches(loser_id, played_at DESC);
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_matches_played ON matches(played_at DESC);
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_matches_active ON matches(deleted_at) WHERE deleted_at IS NULL;
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE TABLE sessions (
      token       TEXT PRIMARY KEY,
      player_id   INTEGER NOT NULL REFERENCES players(id),
      created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
      expires_at  TEXT NOT NULL
  );
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_sessions_player ON sessions(player_id);
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE INDEX idx_sessions_expires ON sessions(expires_at);
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE TABLE admin_sessions (
      token       TEXT PRIMARY KEY,
      created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
      expires_at  TEXT NOT NULL
  );
  -- +goose StatementEnd

  -- +goose StatementBegin
  CREATE TABLE settings (
      key         TEXT PRIMARY KEY,
      value       TEXT NOT NULL
  );
  -- +goose StatementEnd

  -- +goose Down
  -- +goose StatementBegin
  DROP TABLE IF EXISTS settings;
  -- +goose StatementEnd

  -- +goose StatementBegin
  DROP TABLE IF EXISTS admin_sessions;
  -- +goose StatementEnd

  -- +goose StatementBegin
  DROP TABLE IF EXISTS sessions;
  -- +goose StatementEnd

  -- +goose StatementBegin
  DROP TABLE IF EXISTS matches;
  -- +goose StatementEnd

  -- +goose StatementBegin
  DROP TABLE IF EXISTS players;
  -- +goose StatementEnd
  ```

- [ ] 用 goose CLI 对一个临时库手动验证迁移可正向、可回滚（驱动用 sqlite3；goose 的 `sqlite3` dialect 与 modernc 在迁移 DDL 上等价。在 `server/` 目录执行）：
  ```bash
  goose -dir internal/db/migrations sqlite3 ./_tmp_migrate_check.db up
  goose -dir internal/db/migrations sqlite3 ./_tmp_migrate_check.db status
  goose -dir internal/db/migrations sqlite3 ./_tmp_migrate_check.db down
  ```
  - `up` 预期输出含 `OK    00001_init.sql`；
  - `status` 预期 `00001_init.sql` 行显示已应用的时间戳（非 `Pending`）；
  - `down` 预期输出含 `OK    00001_init.sql`（回滚成功）。
  验证完成后删除临时库：
  ```bash
  rm -f ./_tmp_migrate_check.db
  ```
  （Windows PowerShell 下用 `Remove-Item ./_tmp_migrate_check.db -ErrorAction SilentlyContinue`。）

- [ ] 提交：
  ```bash
  git add server/internal/db/migrations/00001_init.sql
  git commit -m "feat(db): add goose 00001_init migration with full schema, extra CHECKs and idx_sessions_expires"
  ```

### Task 2.2: sqlc 配置（sqlc.yaml）

**Files:**
- Create: `server/sqlc.yaml`
- Test: 通过 Task 2.4 的 `sqlc generate` 成功执行 + 生成代码可编译验证

> engine sqlite；queries 指向 `queries`；schema 指向 migrations；输出到 `internal/db/sqlc`，包名 `sqlc`；`emit_json_tags`（JSON 为 snake_case，前端直接对齐）+ `emit_interface`（生成 `Querier`）。

步骤：

- [ ] 创建 `server/sqlc.yaml`，完整内容（version 2 格式；路径相对 `server/`）：
  ```yaml
  version: "2"
  sql:
    - engine: "sqlite"
      schema: "internal/db/migrations"
      queries: "queries"
      gen:
        go:
          package: "sqlc"
          out: "internal/db/sqlc"
          emit_json_tags: true
          emit_interface: true
          emit_empty_slices: true
          overrides:
            - db_type: "TEXT"
              go_type: "string"
  ```
  说明：`schema` 直接指向 goose 迁移目录 —— sqlc 会忽略 `-- +goose` 注释并解析其中的 DDL。`emit_empty_slices` 让 `:many` 查询无结果时返回 `[]T{}` 而非 `nil`，便于上层与 JSON 序列化（前端拿到 `[]` 而非 `null`）。`overrides` 把 `TEXT` 列映射为 Go `string`（时间列我们以 RFC3339 字符串读写，在 service 层做 parse）。

- [ ] 提交：
  ```bash
  git add server/sqlc.yaml
  git commit -m "feat(db): add sqlc.yaml (sqlite engine, json tags, Querier interface)"
  ```

### Task 2.3: 查询 SQL 文件（queries/*.sql）

**Files:**
- Create: `server/queries/players.sql`
- Create: `server/queries/matches.sql`
- Create: `server/queries/sessions.sql`
- Create: `server/queries/settings.sql`
- Test: 通过 Task 2.4 的 `sqlc generate` 验证（查询名、参数与列必须能被 sqlc 解析为类型安全代码）

> 必须**包含且仅用**契约表格列出的查询名。查询按主题拆分到 4 个文件。每个查询用 `-- name: <Name> :<kind>` 注释。

步骤：

- [ ] 创建 `server/queries/players.sql`，完整内容：
  ```sql
  -- name: CreatePlayer :one
  INSERT INTO players (username, rating)
  VALUES (?, ?)
  RETURNING *;

  -- name: GetPlayerByID :one
  SELECT * FROM players
  WHERE id = ?;

  -- name: GetPlayerByUsername :one
  SELECT * FROM players
  WHERE username = ?;

  -- name: ListPlayersByRating :many
  SELECT * FROM players
  ORDER BY rating DESC, id ASC;

  -- name: UpdatePlayerRating :exec
  UPDATE players
  SET rating = ?
  WHERE id = ?;
  ```
  说明：`GetPlayerByUsername` 的 NOCASE 大小写不敏感由列定义 `COLLATE NOCASE` 保证，查询本身无需特殊处理。

- [ ] 创建 `server/queries/matches.sql`，完整内容（`CreateMatch` 显式列出所有快照字段；时间列以字符串传入；`GetPlayerHistory` 返回契约指定的 5 列）：
  ```sql
  -- name: CreateMatch :one
  INSERT INTO matches (
      winner_id, loser_id, submitter_id,
      winner_rating_before, loser_rating_before,
      winner_rating_after, loser_rating_after,
      winner_delta, loser_delta,
      played_at, created_at
  ) VALUES (
      ?, ?, ?,
      ?, ?,
      ?, ?,
      ?, ?,
      ?, ?
  )
  RETURNING *;

  -- name: GetMatchByID :one
  SELECT * FROM matches
  WHERE id = ?;

  -- name: ListGlobalMatches :many
  SELECT * FROM matches
  WHERE deleted_at IS NULL
  ORDER BY played_at DESC, id DESC
  LIMIT ? OFFSET ?;

  -- name: ListPlayerMatches :many
  SELECT * FROM matches
  WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL
  ORDER BY played_at DESC, id DESC
  LIMIT ? OFFSET ?;

  -- name: GetPlayerHistory :many
  SELECT played_at, winner_id, loser_id, winner_rating_after, loser_rating_after
  FROM matches
  WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL
  ORDER BY played_at ASC, id ASC;

  -- name: SoftDeleteMatch :exec
  UPDATE matches
  SET deleted_at = ?, deleted_by = ?
  WHERE id = ? AND deleted_at IS NULL;

  -- name: RestoreMatch :exec
  UPDATE matches
  SET deleted_at = NULL, deleted_by = NULL
  WHERE id = ?;

  -- name: ListDeletedMatches :many
  SELECT * FROM matches
  WHERE deleted_at IS NOT NULL
  ORDER BY deleted_at DESC;

  -- name: CountPlayerWinsLosses :one
  SELECT
      COALESCE(SUM(CASE WHEN winner_id = ? THEN 1 ELSE 0 END), 0) AS wins,
      COALESCE(SUM(CASE WHEN loser_id  = ? THEN 1 ELSE 0 END), 0) AS losses
  FROM matches
  WHERE (winner_id = ? OR loser_id = ?) AND deleted_at IS NULL;
  ```
  说明：`CountPlayerWinsLosses` 用 4 个 `?` 占位（wins 的 winner_id、losses 的 loser_id、WHERE 的 winner_id/loser_id）；service 层调用时把同一个 player_id 传 4 次（顺序与占位一致）。`COALESCE` 保证零局时返回 0 而非 NULL。

- [ ] 创建 `server/queries/sessions.sql`，完整内容（玩家会话、管理员会话、过期清理；时间列以字符串传入）：
  ```sql
  -- name: CreateSession :exec
  INSERT INTO sessions (token, player_id, created_at, expires_at)
  VALUES (?, ?, ?, ?);

  -- name: GetSession :one
  SELECT * FROM sessions
  WHERE token = ?;

  -- name: DeleteSession :exec
  DELETE FROM sessions
  WHERE token = ?;

  -- name: DeleteExpiredSessions :exec
  DELETE FROM sessions
  WHERE expires_at <= ?;

  -- name: CreateAdminSession :exec
  INSERT INTO admin_sessions (token, created_at, expires_at)
  VALUES (?, ?, ?);

  -- name: GetAdminSession :one
  SELECT * FROM admin_sessions
  WHERE token = ?;

  -- name: DeleteAdminSession :exec
  DELETE FROM admin_sessions
  WHERE token = ?;
  ```

- [ ] 创建 `server/queries/settings.sql`，完整内容（`SetSetting` 用 UPSERT）：
  ```sql
  -- name: GetSetting :one
  SELECT value FROM settings
  WHERE key = ?;

  -- name: SetSetting :exec
  INSERT INTO settings (key, value)
  VALUES (?, ?)
  ON CONFLICT(key) DO UPDATE SET value = excluded.value;
  ```

- [ ] 提交：
  ```bash
  git add server/queries/players.sql server/queries/matches.sql server/queries/sessions.sql server/queries/settings.sql
  git commit -m "feat(db): add all sqlc query files per contract query names"
  ```

### Task 2.4: 生成 sqlc 代码并验证可编译

**Files:**
- Create（由工具生成）：`server/internal/db/sqlc/db.go`、`server/internal/db/sqlc/models.go`、`server/internal/db/sqlc/querier.go`、`server/internal/db/sqlc/players.sql.go`、`server/internal/db/sqlc/matches.sql.go`、`server/internal/db/sqlc/sessions.sql.go`、`server/internal/db/sqlc/settings.sql.go`
- Test: `go build ./...` 验证生成代码可编译

步骤：

- [ ] 在 `server/` 目录执行 sqlc 生成（读取 `sqlc.yaml`）：
  ```bash
  sqlc generate
  ```
  预期：无输出、退出码 0。生成后 `internal/db/sqlc/` 下出现上列 `.go` 文件，其中 `querier.go` 含 `Querier` 接口（因 `emit_interface: true`），`db.go` 含 `Queries` 结构体与 `New(db DBTX) *Queries`。

- [ ] 确认生成代码可编译（生成物会引用 `database/sql` 与 modernc 驱动间接接口，但本身不 import 驱动）：
  ```bash
  go build ./...
  ```
  预期：无输出、退出码 0。

- [ ] 确认整树测试仍可扫描（domain 包测试应继续通过）：
  ```bash
  go test ./...
  ```
  预期：`ok  	go_ultra/internal/domain ...`，其余无测试包显示 `no test files`，无 FAIL。

- [ ] 提交生成代码（将生成物纳入版本控制，免去 CI 安装 sqlc）：
  ```bash
  git add server/internal/db/sqlc/
  git commit -m "feat(db): generate sqlc type-safe queries (run: sqlc generate)"
  ```

### Task 2.5: db.New 连接与自动迁移 + 集成测试

**Files:**
- Create: `server/internal/db/db.go`
- Test: `server/internal/db/db_test.go`

> `db.New(path string) (*sql.DB, error)`：用 modernc 驱动（名 `sqlite`）打开；设 `PRAGMA foreign_keys=ON, journal_mode=WAL, busy_timeout=5000`；用 goose 跑迁移。迁移源用 Go embed 嵌入 migrations 目录，使二进制自带迁移、无需运行时找文件。集成测试用临时文件库运行 `New` 后断言全部表存在。

步骤：

- [ ] 先写失败测试 `server/internal/db/db_test.go`，完整内容（用 `t.TempDir()` 建临时库；调用 `New`；查 `sqlite_master` 断言每张表存在；并验证一个 CHECK 约束真正生效 —— 插入 winner==loser 应被拒绝）：
  ```go
  package db

  import (
  	"database/sql"
  	"path/filepath"
  	"testing"
  )

  func TestNewRunsMigrationsAndCreatesTables(t *testing.T) {
  	dbPath := filepath.Join(t.TempDir(), "test.db")
  	database, err := New(dbPath)
  	if err != nil {
  		t.Fatalf("New(%q) error: %v", dbPath, err)
  	}
  	defer database.Close()

  	wantTables := []string{"players", "matches", "sessions", "admin_sessions", "settings"}
  	for _, name := range wantTables {
  		var got string
  		err := database.QueryRow(
  			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
  		).Scan(&got)
  		if err != nil {
  			t.Fatalf("table %q not found after migration: %v", name, err)
  		}
  		if got != name {
  			t.Fatalf("expected table %q, got %q", name, got)
  		}
  	}
  }

  func TestNewSetsForeignKeysPragma(t *testing.T) {
  	dbPath := filepath.Join(t.TempDir(), "test.db")
  	database, err := New(dbPath)
  	if err != nil {
  		t.Fatalf("New error: %v", err)
  	}
  	defer database.Close()

  	var fk int
  	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
  		t.Fatalf("query PRAGMA foreign_keys: %v", err)
  	}
  	if fk != 1 {
  		t.Fatalf("foreign_keys pragma = %d, want 1", fk)
  	}

  	var mode string
  	if err := database.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
  		t.Fatalf("query PRAGMA journal_mode: %v", err)
  	}
  	if mode != "wal" {
  		t.Fatalf("journal_mode = %q, want wal", mode)
  	}
  }

  func TestNewEnforcesSelfMatchCheck(t *testing.T) {
  	dbPath := filepath.Join(t.TempDir(), "test.db")
  	database, err := New(dbPath)
  	if err != nil {
  		t.Fatalf("New error: %v", err)
  	}
  	defer database.Close()

  	// 先建一个玩家以满足外键
  	if _, err := database.Exec(
  		`INSERT INTO players (username, rating) VALUES ('alice', 1500)`,
  	); err != nil {
  		t.Fatalf("insert player: %v", err)
  	}

  	// winner_id == loser_id 应触发 CHECK 失败
  	_, err = insertSelfMatch(database)
  	if err == nil {
  		t.Fatalf("expected CHECK(winner_id != loser_id) to reject self-match, got nil error")
  	}
  }

  func insertSelfMatch(database *sql.DB) (sql.Result, error) {
  	return database.Exec(`
  		INSERT INTO matches (
  			winner_id, loser_id, submitter_id,
  			winner_rating_before, loser_rating_before,
  			winner_rating_after, loser_rating_after,
  			winner_delta, loser_delta,
  			played_at, created_at
  		) VALUES (
  			1, 1, 1,
  			1500, 1500,
  			1508, 1492,
  			8, -8,
  			'2026-06-25T00:00:00Z', '2026-06-25T00:00:00Z'
  		)`)
  }
  ```

- [ ] 运行确认失败（`New` 与 embed 尚不存在，编译失败）：
  ```bash
  go test ./internal/db/
  ```
  预期失败信息形如：`undefined: New`，`FAIL go_ultra/internal/db [build failed]`。

- [ ] 写最小实现 `server/internal/db/db.go`，完整内容（用 `embed` 嵌入 migrations；goose 的 `SetBaseFS` + `SetDialect("sqlite3")` + `Up`；modernc 驱动 import 时以 `_` 注册，驱动名为 `sqlite`）：
  ```go
  package db

  import (
  	"database/sql"
  	"embed"
  	"fmt"

  	"github.com/pressly/goose/v3"
  	_ "modernc.org/sqlite"
  )

  //go:embed migrations/*.sql
  var migrationsFS embed.FS

  // New 打开 SQLite 数据库，设置 PRAGMA，运行 goose 迁移，返回 *sql.DB。
  func New(path string) (*sql.DB, error) {
  	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
  	database, err := sql.Open("sqlite", dsn)
  	if err != nil {
  		return nil, fmt.Errorf("open sqlite: %w", err)
  	}

  	if err := database.Ping(); err != nil {
  		database.Close()
  		return nil, fmt.Errorf("ping sqlite: %w", err)
  	}

  	if err := runMigrations(database); err != nil {
  		database.Close()
  		return nil, fmt.Errorf("run migrations: %w", err)
  	}

  	return database, nil
  }

  func runMigrations(database *sql.DB) error {
  	goose.SetBaseFS(migrationsFS)
  	if err := goose.SetDialect("sqlite3"); err != nil {
  		return fmt.Errorf("set goose dialect: %w", err)
  	}
  	if err := goose.Up(database, "migrations"); err != nil {
  		return fmt.Errorf("goose up: %w", err)
  	}
  	return nil
  }
  ```
  说明：modernc 驱动通过 DSN query 参数 `_pragma=...` 设置 PRAGMA（modernc 支持 `_pragma` 语法），保证每个连接都带上这些 PRAGMA（连接池下尤其重要），比单次 `Exec("PRAGMA ...")` 更可靠。

- [ ] 运行确认通过：
  ```bash
  go test ./internal/db/
  ```
  预期：`ok  	go_ultra/internal/db`。

- [ ] 跑带竞态检测的全树测试，确认无并发问题、无回归：
  ```bash
  go test -race ./...
  ```
  预期：`ok  	go_ultra/internal/db`、`ok  	go_ultra/internal/domain`，其余 `no test files`，无 FAIL。

- [ ] 提交：
  ```bash
  git add server/internal/db/db.go server/internal/db/db_test.go
  git commit -m "feat(db): add db.New with embedded migrations, PRAGMAs and integration test"
  ```

---

## 阶段 3: service 层

> 本阶段在 `server/internal/service` 下实现四个业务服务：`PlayerService`、`MatchService`、`LeaderboardService`、`AdminService`，并为每个服务配一个**真实数据库集成测试**（modernc sqlite 临时 `.db` 文件 + goose 迁移 + sqlc 生成的 `Queries`）。
>
> **本阶段的前置假设**（由阶段 1、2 完成，已是既成事实，不在本阶段重复实现）：
> - `go_ultra/internal/domain`：已存在 `types.go`（`Player`/`Stats`/`Match`）、`elo.go`（`DefaultRating=1500`、`KFactor=16`、`ExpectedScore`、`ComputeDelta`）、`rank.go`（`RankFloor=1050`、`Dan`）、`errors.go`（`*Error` 类型与 `ErrPlayerNotFound`/`ErrMatchNotFound`/`ErrSelfMatch`/`ErrInternal` 等预定义错误、`WithCause`）。
> - `go_ultra/internal/db`：已存在 `db.New(path string) (*sql.DB, error)`，它打开 modernc sqlite、设 `PRAGMA foreign_keys=ON, journal_mode=WAL, busy_timeout=5000`、运行 `00001_init.sql` 的 goose Up，返回 `*sql.DB`。
> - `go_ultra/internal/db/sqlc`：sqlc v1.27 生成包，包名 `sqlc`，含 `New(db sqlc.DBTX) *Queries`、`Queries` 结构体（含 `WithTx(tx *sql.Tx) *Queries`）、`Querier` 接口，以及契约中列出的全部查询方法。本阶段集成测试用 `db.New` 打开临时 `.db` 拿到 `*sql.DB`，再用 `sqlc.New(sqlDB)` 构造 `*sqlc.Queries`。
> - `go_ultra/internal/session`：已存在 `NewToken() (string, error)`、`AdminSessionTTL = 30 * time.Minute`。
>
> **sqlc 生成类型命名约定**（sqlc v1.27 默认规则，本阶段代码严格依赖）：
> - 行模型：`sqlc.Player`、`sqlc.Match`、`sqlc.AdminSession`、`sqlc.Setting` 等，字段为大驼峰（`Username`、`Rating`、`WinnerID`、`DeletedAt` 等），可空列为指针或 `sql.NullXxx`。
> - 多参数查询生成 `<QueryName>Params` 结构体（如 `CreatePlayerParams`、`CreateMatchParams`、`SoftDeleteMatchParams`）。
> - 自定义投影查询生成 `<QueryName>Row`（如 `CountPlayerWinsLossesRow`、`GetPlayerHistoryRow`）。
> - 时间列为 `TEXT`（RFC3339 字符串），sqlc 生成为 `string` 字段；service 层负责 `time.Time ↔ RFC3339 string` 的转换，统一用 `time.RFC3339`、`time.Now().UTC()`。
>
> **公共转换约定**（本阶段在 `service/convert.go` 集中实现一次，四个服务共用）：把 RFC3339 字符串解析为 `time.Time`、把 `time.Time` 格式化为 RFC3339 字符串、把 `sqlc.Player` 映射为 `domain.Player`、把 `sqlc.Match` 映射为 `domain.Match`。

---

### Task 1: service 公共转换层 `convert.go`

四个服务都需要在 `domain` 类型、`time.Time` 和 sqlc 的字符串/可空类型之间转换。先把这些纯函数用 TDD 落地，后续服务直接复用，避免重复。

**Files:**
- Create: `server/internal/service/convert.go`
- Test: `server/internal/service/convert_test.go`

步骤：

- [ ] 写失败测试。创建 `server/internal/service/convert_test.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"database/sql"
	"testing"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

func TestParseTime(t *testing.T) {
	want := time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)
	got, err := parseTime("2026-06-25T14:30:00Z")
	if err != nil {
		t.Fatalf("parseTime error: %v", err)
	}
	if !got.Equal(want) {
		t.Fatalf("parseTime = %v, want %v", got, want)
	}
	if _, err := parseTime("not-a-time"); err == nil {
		t.Fatalf("parseTime(bad) expected error, got nil")
	}
}

func TestFormatTime(t *testing.T) {
	in := time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)
	if got := formatTime(in); got != "2026-06-25T14:30:00Z" {
		t.Fatalf("formatTime = %q, want %q", got, "2026-06-25T14:30:00Z")
	}
	// 非 UTC 输入必须被规整为 UTC 再格式化
	loc := time.FixedZone("X", 3600)
	in2 := time.Date(2026, 6, 25, 15, 30, 0, 0, loc) // == 14:30Z
	if got := formatTime(in2); got != "2026-06-25T14:30:00Z" {
		t.Fatalf("formatTime(non-utc) = %q, want %q", got, "2026-06-25T14:30:00Z")
	}
}

func TestToDomainPlayer(t *testing.T) {
	row := sqlc.Player{
		ID:        7,
		Username:  "alice",
		Rating:    1500,
		CreatedAt: "2026-06-25T14:30:00Z",
	}
	p, err := toDomainPlayer(row)
	if err != nil {
		t.Fatalf("toDomainPlayer error: %v", err)
	}
	if p.ID != 7 || p.Username != "alice" || p.Rating != 1500 {
		t.Fatalf("toDomainPlayer scalar mismatch: %+v", p)
	}
	if !p.CreatedAt.Equal(time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC)) {
		t.Fatalf("toDomainPlayer time mismatch: %v", p.CreatedAt)
	}
}

func TestToDomainMatch(t *testing.T) {
	del := "2026-06-26T00:00:00Z"
	row := sqlc.Match{
		ID:                 3,
		WinnerID:           1,
		LoserID:            2,
		SubmitterID:        1,
		WinnerRatingBefore: 1500,
		LoserRatingBefore:  1500,
		WinnerRatingAfter:  1508,
		LoserRatingAfter:   1492,
		WinnerDelta:        8,
		LoserDelta:         -8,
		PlayedAt:           "2026-06-25T14:30:00Z",
		CreatedAt:          "2026-06-25T14:30:01Z",
		DeletedAt:          sql.NullString{String: del, Valid: true},
		DeletedBy:          sql.NullInt64{Valid: false},
	}
	m, err := toDomainMatch(row)
	if err != nil {
		t.Fatalf("toDomainMatch error: %v", err)
	}
	if m.ID != 3 || m.WinnerID != 1 || m.LoserID != 2 || m.WinnerDelta != 8 || m.LoserDelta != -8 {
		t.Fatalf("toDomainMatch scalar mismatch: %+v", m)
	}
	if m.DeletedAt == nil || !m.DeletedAt.Equal(time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("toDomainMatch DeletedAt mismatch: %v", m.DeletedAt)
	}
	if m.DeletedBy != nil {
		t.Fatalf("toDomainMatch DeletedBy should be nil, got %v", *m.DeletedBy)
	}
}

// 编译期保证 domain 包被使用（避免误删 import）
var _ = domain.DefaultRating
```

- [ ] 运行确认失败：

```
cd server && go test ./internal/service/ -run 'TestParseTime|TestFormatTime|TestToDomainPlayer|TestToDomainMatch'
```

预期失败：编译错误，形如 `undefined: parseTime`、`undefined: formatTime`、`undefined: toDomainPlayer`、`undefined: toDomainMatch`（`convert.go` 尚不存在）。

- [ ] 写最小实现。创建 `server/internal/service/convert.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"database/sql"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

// parseTime 把 RFC3339 字符串解析为 UTC time.Time。
func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// formatTime 把 time.Time 规整为 UTC 后格式化为 RFC3339 字符串。
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// nullStringTimePtr 把可空时间列转换为 *time.Time。
func nullStringTimePtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// nullInt64Ptr 把可空整型列转换为 *int64。
func nullInt64Ptr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	v := ni.Int64
	return &v
}

// toDomainPlayer 把 sqlc.Player 行映射为 domain.Player。
func toDomainPlayer(p sqlc.Player) (domain.Player, error) {
	createdAt, err := parseTime(p.CreatedAt)
	if err != nil {
		return domain.Player{}, err
	}
	return domain.Player{
		ID:        p.ID,
		Username:  p.Username,
		Rating:    int(p.Rating),
		CreatedAt: createdAt,
	}, nil
}

// toDomainMatch 把 sqlc.Match 行映射为 domain.Match。
func toDomainMatch(m sqlc.Match) (domain.Match, error) {
	playedAt, err := parseTime(m.PlayedAt)
	if err != nil {
		return domain.Match{}, err
	}
	createdAt, err := parseTime(m.CreatedAt)
	if err != nil {
		return domain.Match{}, err
	}
	deletedAt, err := nullStringTimePtr(m.DeletedAt)
	if err != nil {
		return domain.Match{}, err
	}
	return domain.Match{
		ID:                 m.ID,
		WinnerID:           m.WinnerID,
		LoserID:            m.LoserID,
		SubmitterID:        m.SubmitterID,
		WinnerRatingBefore: int(m.WinnerRatingBefore),
		LoserRatingBefore:  int(m.LoserRatingBefore),
		WinnerRatingAfter:  int(m.WinnerRatingAfter),
		LoserRatingAfter:   int(m.LoserRatingAfter),
		WinnerDelta:        int(m.WinnerDelta),
		LoserDelta:         int(m.LoserDelta),
		PlayedAt:           playedAt,
		CreatedAt:          createdAt,
		DeletedAt:          deletedAt,
		DeletedBy:          nullInt64Ptr(m.DeletedBy),
	}, nil
}
```

> 说明：sqlc 把 `rating`/`winner_rating_before` 等 `INTEGER` 列生成为 `int64`，而 `domain` 用 `int`，故映射处显式 `int(...)` 转换。若你所在仓库的 sqlc 配置把 `INTEGER` 直接生成为 `int`，去掉这些 `int(...)` 包裹即可（编译器会立即提示 `redundant conversion`，按提示删除）。

- [ ] 运行确认通过：

```
cd server && go test ./internal/service/ -run 'TestParseTime|TestFormatTime|TestToDomainPlayer|TestToDomainMatch'
```

预期输出包含 `ok  	go_ultra/internal/service`。

- [ ] commit：

```
git add server/internal/service/convert.go server/internal/service/convert_test.go
git commit -m "feat(service): add domain/time conversion helpers"
```

---

### Task 2: 集成测试公共脚手架 `helpers_test.go`

四个服务的集成测试都需要：开一个临时 `.db`、跑迁移、拿到 `*sql.DB` 与 `*sqlc.Queries`、测试结束清理。把它做成一个共享的测试辅助，避免每个测试文件重复。

**Files:**
- Create:（无生产代码，仅测试辅助）
- Test: `server/internal/service/helpers_test.go`

步骤：

- [ ] 写测试（这次"测试"本身就是验证脚手架可用）。创建 `server/internal/service/helpers_test.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"go_ultra/internal/db"
	"go_ultra/internal/db/sqlc"
)

// newTestDB 在测试临时目录开一个全新的 sqlite 文件，跑迁移，返回 *sql.DB 与 *sqlc.Queries。
// 测试结束自动关闭连接（临时目录由 testing 框架清理）。
func newTestDB(t *testing.T) (*sql.DB, *sqlc.Queries) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	sqlDB, err := db.New(path)
	if err != nil {
		t.Fatalf("db.New(%q) error: %v", path, err)
	}
	t.Cleanup(func() {
		if cerr := sqlDB.Close(); cerr != nil {
			t.Errorf("close db: %v", cerr)
		}
	})
	return sqlDB, sqlc.New(sqlDB)
}

// ctx 返回带超时的测试上下文，超时后自动取消。
func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// mustCreatePlayer 直接经 Queries 建一个玩家，返回其 ID（绕过 service，仅用于布置测试前置数据）。
func mustCreatePlayer(t *testing.T, q *sqlc.Queries, username string, rating int) int64 {
	t.Helper()
	p, err := q.CreatePlayer(context.Background(), sqlc.CreatePlayerParams{
		Username: username,
		Rating:   int64(rating),
	})
	if err != nil {
		t.Fatalf("CreatePlayer(%q) error: %v", username, err)
	}
	return p.ID
}

// TestScaffold 自检：确认临时库能开、迁移能跑、能建玩家。
func TestScaffold(t *testing.T) {
	q, queries := newTestDB(t), (*sqlc.Queries)(nil)
	_ = q
	sqlDB, qq := newTestDB(t)
	queries = qq
	if sqlDB == nil || queries == nil {
		t.Fatal("newTestDB returned nil")
	}
	id := mustCreatePlayer(t, queries, "scaffold_user", 1500)
	if id <= 0 {
		t.Fatalf("expected positive player id, got %d", id)
	}
	got, err := queries.GetPlayerByID(testCtx(t), id)
	if err != nil {
		t.Fatalf("GetPlayerByID error: %v", err)
	}
	if got.Username != "scaffold_user" || got.Rating != 1500 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
```

> 说明：`CreatePlayerParams.Rating` 为 sqlc 生成的 `int64`，故传入 `int64(rating)`。如果你的 sqlc 配置把该列生成为 `int`，编译器会报 `cannot use int64(rating) ... as int`，把 `int64(rating)` 改成 `rating` 即可。`GetPlayerByID` 的 schema 由迁移保证。

- [ ] 运行确认通过（此处脚手架本应直接通过，因为依赖的 `db.New`、`sqlc` 已由前置阶段提供）：

```
cd server && go test ./internal/service/ -run TestScaffold -v
```

预期输出包含 `--- PASS: TestScaffold` 与 `ok  	go_ultra/internal/service`。
若失败信息为 `undefined: db.New` 或 `undefined: sqlc.CreatePlayerParams`，说明前置阶段（阶段 1/2）尚未合入，必须先完成它们再继续本阶段。

- [ ] commit：

```
git add server/internal/service/helpers_test.go
git commit -m "test(service): add integration test scaffolding (temp sqlite + goose + sqlc)"
```

---

### Task 3: `PlayerService`

实现 `LoginOrCreate`（trim + 校验 3–32 字符；存在则返回，否则按 `DefaultRating` 创建，幂等）、`GetByUsername`、`GetStats`（用 `CountPlayerWinsLosses` 取胜负，再遍历该玩家全部未删除对局算 current/longest streak）、`ListByRating`。

**Files:**
- Create: `server/internal/service/player.go`
- Test: `server/internal/service/player_test.go`

步骤：

- [ ] 写失败测试。创建 `server/internal/service/player_test.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

func TestPlayerService_LoginOrCreate_CreatesWithDefaultRating(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	p, err := svc.LoginOrCreate(ctx, "  Alice  ") // 含空白，必须 trim
	if err != nil {
		t.Fatalf("LoginOrCreate error: %v", err)
	}
	if p.Username != "Alice" {
		t.Fatalf("username not trimmed: %q", p.Username)
	}
	if p.Rating != domain.DefaultRating {
		t.Fatalf("rating = %d, want %d", p.Rating, domain.DefaultRating)
	}
	if p.ID <= 0 {
		t.Fatalf("expected positive id, got %d", p.ID)
	}
}

func TestPlayerService_LoginOrCreate_Idempotent(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	first, err := svc.LoginOrCreate(ctx, "bob")
	if err != nil {
		t.Fatalf("first LoginOrCreate error: %v", err)
	}
	// 第二次（大小写不同，依赖 username 列 COLLATE NOCASE）必须复用同一行
	second, err := svc.LoginOrCreate(ctx, "BOB")
	if err != nil {
		t.Fatalf("second LoginOrCreate error: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("not idempotent: first id=%d second id=%d", first.ID, second.ID)
	}
	all, err := svc.ListByRating(ctx)
	if err != nil {
		t.Fatalf("ListByRating error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected exactly 1 player, got %d", len(all))
	}
}

func TestPlayerService_LoginOrCreate_Validation(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	cases := []string{
		"",       // 空
		"   ",    // 全空白 trim 后为空
		"ab",     // 2 字符，太短
		string(make([]byte, 33)), // 33 字节，太长（占位长度）
	}
	for _, name := range cases {
		if _, err := svc.LoginOrCreate(ctx, name); err == nil {
			t.Fatalf("LoginOrCreate(%q) expected error, got nil", name)
		}
	}
	// 边界：恰好 3 与恰好 32 必须成功
	if _, err := svc.LoginOrCreate(ctx, "abc"); err != nil {
		t.Fatalf("LoginOrCreate(3 chars) error: %v", err)
	}
	name32 := "abcdefghijklmnopqrstuvwxyz012345" // 32 字符
	if len([]rune(name32)) != 32 {
		t.Fatalf("test fixture wrong length: %d", len([]rune(name32)))
	}
	if _, err := svc.LoginOrCreate(ctx, name32); err != nil {
		t.Fatalf("LoginOrCreate(32 chars) error: %v", err)
	}
}

func TestPlayerService_GetByUsername(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	created, err := svc.LoginOrCreate(ctx, "carol")
	if err != nil {
		t.Fatalf("LoginOrCreate error: %v", err)
	}
	got, err := svc.GetByUsername(ctx, "carol")
	if err != nil {
		t.Fatalf("GetByUsername error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id mismatch: %d vs %d", got.ID, created.ID)
	}
	// 不存在 → ErrPlayerNotFound
	_, err = svc.GetByUsername(ctx, "nobody")
	if !errors.Is(err, domain.ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}

// insertMatch 直接经 Queries 录入一条对局（绕过 service，用于精确布置 streak 数据）。
func insertMatch(t *testing.T, q *sqlc.Queries, winnerID, loserID, submitterID int64, playedAt time.Time) {
	t.Helper()
	_, err := q.CreateMatch(context.Background(), sqlc.CreateMatchParams{
		WinnerID:           winnerID,
		LoserID:            loserID,
		SubmitterID:        submitterID,
		WinnerRatingBefore: 1500,
		LoserRatingBefore:  1500,
		WinnerRatingAfter:  1508,
		LoserRatingAfter:   1492,
		WinnerDelta:        8,
		LoserDelta:         -8,
		PlayedAt:           formatTime(playedAt),
		CreatedAt:          formatTime(playedAt),
	})
	if err != nil {
		t.Fatalf("CreateMatch error: %v", err)
	}
}

func TestPlayerService_GetStats_StreaksAndWinRate(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	a := mustCreatePlayer(t, q, "streak_a", 1500)
	b := mustCreatePlayer(t, q, "streak_b", 1500)

	base := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	// 时间升序录入：a 的结果序列为 W W L W W W（最后三连胜）
	// played_at 递增，确保历史遍历顺序确定
	insertMatch(t, q, a, b, a, base.Add(1*time.Minute)) // a win
	insertMatch(t, q, a, b, a, base.Add(2*time.Minute)) // a win
	insertMatch(t, q, b, a, b, base.Add(3*time.Minute)) // a loss
	insertMatch(t, q, a, b, a, base.Add(4*time.Minute)) // a win
	insertMatch(t, q, a, b, a, base.Add(5*time.Minute)) // a win
	insertMatch(t, q, a, b, a, base.Add(6*time.Minute)) // a win

	stats, err := svc.GetStats(ctx, a)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if stats.Wins != 5 || stats.Losses != 1 {
		t.Fatalf("wins/losses = %d/%d, want 5/1", stats.Wins, stats.Losses)
	}
	wantRate := 5.0 / 6.0
	if stats.WinRate < wantRate-1e-9 || stats.WinRate > wantRate+1e-9 {
		t.Fatalf("win rate = %v, want %v", stats.WinRate, wantRate)
	}
	if stats.CurrentStreak != 3 {
		t.Fatalf("current streak = %d, want 3", stats.CurrentStreak)
	}
	if stats.LongestStreak != 3 {
		t.Fatalf("longest streak = %d, want 3", stats.LongestStreak)
	}
}

func TestPlayerService_GetStats_NoMatches(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	a := mustCreatePlayer(t, q, "lonely", 1500)
	stats, err := svc.GetStats(ctx, a)
	if err != nil {
		t.Fatalf("GetStats error: %v", err)
	}
	if stats.Wins != 0 || stats.Losses != 0 || stats.WinRate != 0 ||
		stats.CurrentStreak != 0 || stats.LongestStreak != 0 {
		t.Fatalf("expected all-zero stats, got %+v", stats)
	}
}

func TestPlayerService_ListByRating_Ordered(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewPlayerService(q, sqlDB)
	ctx := testCtx(t)

	mustCreatePlayer(t, q, "low", 1400)
	mustCreatePlayer(t, q, "high", 1600)
	mustCreatePlayer(t, q, "mid", 1500)

	list, err := svc.ListByRating(ctx)
	if err != nil {
		t.Fatalf("ListByRating error: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 players, got %d", len(list))
	}
	if list[0].Username != "high" || list[1].Username != "mid" || list[2].Username != "low" {
		t.Fatalf("not sorted by rating desc: %v", []string{list[0].Username, list[1].Username, list[2].Username})
	}
}
```

> 关于 streak 语义：current streak = 从该玩家**最近一局往回**连续胜局数（一旦遇到败局即停止；最近一局是败局则为 0）。longest streak = 历史上最长的连续胜局数。两者都只统计该玩家未删除对局，按 `played_at ASC` 遍历。

- [ ] 运行确认失败：

```
cd server && go test ./internal/service/ -run TestPlayerService
```

预期失败：编译错误 `undefined: NewPlayerService`（`player.go` 尚不存在）。

- [ ] 写最小实现。创建 `server/internal/service/player.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
	"go_ultra/internal/session"
)

// PlayerService 负责玩家账号的隐式注册、查询与统计。
type PlayerService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewPlayerService 构造 PlayerService。
func NewPlayerService(q *sqlc.Queries, db *sql.DB) *PlayerService {
	return &PlayerService{q: q, db: db}
}

// validateUsername trim 后校验长度为 3–32 个字符（按 rune 计数）。
func validateUsername(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(name)
	if n < 3 || n > 32 {
		return "", domain.ErrInvalidParam
	}
	return name, nil
}

// LoginOrCreate 校验并 trim 用户名；已存在则返回该玩家，否则按 DefaultRating 创建。幂等。
func (s *PlayerService) LoginOrCreate(ctx context.Context, username string) (domain.Player, error) {
	name, err := validateUsername(username)
	if err != nil {
		return domain.Player{}, err
	}

	// 先查（username 列 COLLATE NOCASE，大小写不敏感）。
	existing, err := s.q.GetPlayerByUsername(ctx, name)
	if err == nil {
		return toDomainPlayer(existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}

	// 不存在 → 创建。
	created, err := s.q.CreatePlayer(ctx, sqlc.CreatePlayerParams{
		Username: name,
		Rating:   int64(domain.DefaultRating),
	})
	if err != nil {
		// 并发下可能撞唯一约束：再查一次，保证幂等。
		again, qerr := s.q.GetPlayerByUsername(ctx, name)
		if qerr == nil {
			return toDomainPlayer(again)
		}
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}
	return toDomainPlayer(created)
}

// GetByUsername 按用户名查玩家；不存在返回 ErrPlayerNotFound。
func (s *PlayerService) GetByUsername(ctx context.Context, username string) (domain.Player, error) {
	name := strings.TrimSpace(username)
	row, err := s.q.GetPlayerByUsername(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Player{}, domain.ErrPlayerNotFound
		}
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}
	return toDomainPlayer(row)
}

// GetStats 统计指定玩家的胜负、胜率与连胜。
func (s *PlayerService) GetStats(ctx context.Context, playerID int64) (domain.Stats, error) {
	counts, err := s.q.CountPlayerWinsLosses(ctx, playerID)
	if err != nil {
		return domain.Stats{}, domain.ErrInternal.WithCause(err)
	}
	wins := int(counts.Wins)
	losses := int(counts.Losses)

	var winRate float64
	total := wins + losses
	if total > 0 {
		winRate = float64(wins) / float64(total)
	}

	// 遍历该玩家全部未删除对局（played_at ASC）算 streak。
	// limit 取一个足够大的常量（朋友圈规模 < 100 人、每天数十局，远不会触顶）。
	history, err := s.q.ListPlayerMatches(ctx, sqlc.ListPlayerMatchesParams{
		WinnerID: playerID,
		LoserID:  playerID,
		Limit:    1000000,
		Offset:   0,
	})
	if err != nil {
		return domain.Stats{}, domain.ErrInternal.WithCause(err)
	}

	// ListPlayerMatches 按 played_at DESC 返回；streak 计算需要时间升序，故倒序遍历。
	current := 0
	longest := 0
	run := 0
	currentDone := false
	for i := len(history) - 1; i >= 0; i-- {
		won := history[i].WinnerID == playerID
		if won {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
		// current streak：从最近一局（升序遍历的最后一条，即原始 i==0）回看。
		// 升序遍历到末尾时 run 即为"最近连胜"。
		_ = currentDone
	}
	// 升序遍历结束后 run 恰好等于"从最近往前的连胜数"（因为最后一段连续胜局未被败局打断）。
	current = run

	return domain.Stats{
		Wins:          wins,
		Losses:        losses,
		WinRate:       winRate,
		CurrentStreak: current,
		LongestStreak: longest,
	}, nil
}

// ListByRating 返回所有玩家，按 rating 降序。
func (s *PlayerService) ListByRating(ctx context.Context) ([]domain.Player, error) {
	rows, err := s.q.ListPlayersByRating(ctx)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	players := make([]domain.Player, 0, len(rows))
	for _, r := range rows {
		p, cerr := toDomainPlayer(r)
		if cerr != nil {
			return nil, domain.ErrInternal.WithCause(cerr)
		}
		players = append(players, p)
	}
	return players, nil
}

// GetByID 按 ID 查玩家；不存在返回 ErrPlayerNotFound。
func (s *PlayerService) GetByID(ctx context.Context, playerID int64) (domain.Player, error) {
	row, err := s.q.GetPlayerByID(ctx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Player{}, domain.ErrPlayerNotFound
		}
		return domain.Player{}, domain.ErrInternal.WithCause(err)
	}
	return toDomainPlayer(row)
}

// CreatePlayerSession 为玩家创建一个会话，返回 token 与过期时间（PlayerSessionTTL）。
func (s *PlayerService) CreatePlayerSession(ctx context.Context, playerID int64) (string, time.Time, error) {
	token, err := session.NewToken()
	if err != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(session.PlayerSessionTTL)
	if err := s.q.CreateSession(ctx, sqlc.CreateSessionParams{
		Token:     token,
		PlayerID:  playerID,
		CreatedAt: formatTime(now),
		ExpiresAt: formatTime(expiresAt),
	}); err != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(err)
	}
	return token, expiresAt, nil
}

// GetSession 校验玩家会话 token；过期或不存在返回 ok=false（非错误）。
func (s *PlayerService) GetSession(ctx context.Context, token string) (int64, bool, error) {
	row, err := s.q.GetSession(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, domain.ErrInternal.WithCause(err)
	}
	expiresAt, perr := parseTime(row.ExpiresAt)
	if perr != nil {
		return 0, false, domain.ErrInternal.WithCause(perr)
	}
	if !time.Now().UTC().Before(expiresAt) {
		// 已过期：惰性删除并视为未认证。
		_ = s.q.DeleteSession(ctx, token)
		return 0, false, nil
	}
	return row.PlayerID, true, nil
}

// DeletePlayerSession 删除指定会话 token（登出）。幂等。
func (s *PlayerService) DeletePlayerSession(ctx context.Context, token string) error {
	if err := s.q.DeleteSession(ctx, token); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}
```

> 会话方法说明：
> - `GetByID` 基于 sqlc `GetPlayerByID`，供 `handleMe` 等通过注入的 `playerID` 取玩家。
> - `CreatePlayerSession` 用 `session.NewToken()` 生成 token，写 `sessions` 表，过期时间为 `now + session.PlayerSessionTTL`（30 天）。
> - `GetSession` 校验未过期则返回 `playerID, true`；过期则惰性 `DeleteSession` 并返回 `ok=false`；`sql.ErrNoRows` 也返回 `ok=false`（中间件据此回 401）。这正是 http 层 `PlayerSessionChecker.GetSession` 接口期望的签名 `(int64, bool, error)`。
> - `DeletePlayerSession` 包装 `DeleteSession`，登出用，幂等。
> - `player.go` 顶部 import 需含 `"time"` 与 `"go_ultra/internal/session"`（已在上面"完整可粘贴"的 import 块中给出；其余 `context`/`database/sql`/`errors`/`strings`/`unicode/utf8`/`sqlc`/`domain` 同块已含）。
> - `CountPlayerWinsLossesRow` 字段名为 `Wins`、`Losses`（sqlc 由 SQL 别名生成）；`ListPlayerMatchesParams` 含 `WinnerID`、`LoserID`、`Limit`、`Offset`（契约：`(winner_id=? OR loser_id=?)`，两个占位符均绑同一 playerID）。
> - `GetPlayerByUsername` 在未命中时 sqlc 返回 `sql.ErrNoRows`，据此区分"不存在"与真实错误。

- [ ] 运行确认通过：

```
cd server && go test ./internal/service/ -run TestPlayerService -v
```

预期输出每个子测试 `--- PASS`，结尾 `ok  	go_ultra/internal/service`。

- [ ] commit：

```
git add server/internal/service/player.go server/internal/service/player_test.go
git commit -m "feat(service): implement PlayerService (login/create, stats, streaks, list)"
```

---

### Task 4: `MatchService`

实现 `Record`（`db.BeginTx` + 事务内全部读写，对应 spec 的 `BEGIN IMMEDIATE` 语义；`result=win` → winner=submitter；查双方 rating；`domain.ComputeDelta`；插入快照完整的 match；更新双方 rating；返回 `RecordResult`；`submitter==opponent` → `domain.ErrSelfMatch`）、`ListGlobal`、`ListByPlayer`（组装 `MatchView`）、`History`（`prepend (createdAt, DefaultRating)`）。

**Files:**
- Create: `server/internal/service/match.go`
- Test: `server/internal/service/match_test.go`

步骤：

- [ ] 写失败测试。创建 `server/internal/service/match_test.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"errors"
	"testing"
	"time"

	"go_ultra/internal/domain"
)

func TestMatchService_Record_WinnerIsSubmitterOnWin(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	_, _ = psvc.LoginOrCreate(ctx, "bob")

	res, err := msvc.Record(ctx, alice.ID, "bob", "win", time.Now().UTC())
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}
	if res.MatchID <= 0 {
		t.Fatalf("expected positive match id, got %d", res.MatchID)
	}
	// 单局零和
	if res.WinnerDelta+res.LoserDelta != 0 {
		t.Fatalf("not zero-sum: winner=%d loser=%d", res.WinnerDelta, res.LoserDelta)
	}
	// 平分对局：delta 应为 round(16*0.5)=8
	if res.WinnerDelta != 8 {
		t.Fatalf("winner delta = %d, want 8 (equal ratings)", res.WinnerDelta)
	}
	if res.NewSelfRating != domain.DefaultRating+res.WinnerDelta {
		t.Fatalf("self rating = %d, want %d", res.NewSelfRating, domain.DefaultRating+res.WinnerDelta)
	}
	if res.NewOpponentRating != domain.DefaultRating+res.LoserDelta {
		t.Fatalf("opponent rating = %d, want %d", res.NewOpponentRating, domain.DefaultRating+res.LoserDelta)
	}
}

func TestMatchService_Record_LossSwapsWinner(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	// alice 提交一场 "loss"：winner 应该是 bob，alice 掉分
	res, err := msvc.Record(ctx, alice.ID, "bob", "loss", time.Now().UTC())
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}
	if res.NewSelfRating >= domain.DefaultRating {
		t.Fatalf("submitter (loser) should lose rating, got %d", res.NewSelfRating)
	}
	if res.NewOpponentRating <= domain.DefaultRating {
		t.Fatalf("opponent (winner) should gain rating, got %d", res.NewOpponentRating)
	}
	if res.WinnerDelta+res.LoserDelta != 0 {
		t.Fatalf("not zero-sum")
	}
}

func TestMatchService_Record_SelfMatch(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	_, err := msvc.Record(ctx, alice.ID, "alice", "win", time.Now().UTC())
	if !errors.Is(err, domain.ErrSelfMatch) {
		t.Fatalf("expected ErrSelfMatch, got %v", err)
	}
}

func TestMatchService_Record_OpponentNotFound(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	_, err := msvc.Record(ctx, alice.ID, "ghost", "win", time.Now().UTC())
	if !errors.Is(err, domain.ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}

func TestMatchService_Record_SumConservedOver100Games(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	bob, _ := psvc.LoginOrCreate(ctx, "bob")

	const initialSum = 2 * domain.DefaultRating
	base := time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		result := "win"
		if i%2 == 1 {
			result = "loss" // 交替胜负，分数来回波动
		}
		_, err := msvc.Record(ctx, alice.ID, "bob", result, base.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatalf("Record #%d error: %v", i, err)
		}
	}
	// 录入 100 局后，两人 rating 之和必须守恒
	pa, err := psvc.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByUsername alice: %v", err)
	}
	pb, err := psvc.GetByUsername(ctx, "bob")
	if err != nil {
		t.Fatalf("GetByUsername bob: %v", err)
	}
	if pa.Rating+pb.Rating != initialSum {
		t.Fatalf("sum not conserved: %d + %d = %d, want %d", pa.Rating, pb.Rating, pa.Rating+pb.Rating, initialSum)
	}
	_ = alice
	_ = bob
}

func TestMatchService_ListGlobal_And_ListByPlayer_View(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	if _, err := msvc.Record(ctx, alice.ID, "bob", "win", at); err != nil {
		t.Fatalf("Record error: %v", err)
	}

	// 全局
	global, err := msvc.ListGlobal(ctx, 50, 0)
	if err != nil {
		t.Fatalf("ListGlobal error: %v", err)
	}
	if len(global) != 1 {
		t.Fatalf("expected 1 global match, got %d", len(global))
	}

	// alice 视角：Result 相对 alice = "win"，Opponent = "bob"
	view, err := msvc.ListByPlayer(ctx, alice.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListByPlayer error: %v", err)
	}
	if len(view) != 1 {
		t.Fatalf("expected 1 match for alice, got %d", len(view))
	}
	mv := view[0]
	if mv.Opponent != "bob" {
		t.Fatalf("opponent = %q, want bob", mv.Opponent)
	}
	if mv.Result != "win" {
		t.Fatalf("result = %q, want win", mv.Result)
	}
	if mv.RatingBefore != domain.DefaultRating {
		t.Fatalf("rating before = %d, want %d", mv.RatingBefore, domain.DefaultRating)
	}
	if mv.RatingAfter != mv.RatingBefore+mv.Delta {
		t.Fatalf("rating math broken: before=%d after=%d delta=%d", mv.RatingBefore, mv.RatingAfter, mv.Delta)
	}
	if mv.Delta <= 0 {
		t.Fatalf("winner delta should be positive, got %d", mv.Delta)
	}
}

func TestMatchService_ListByPlayer_LoserPerspective(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	bob, _ := psvc.LoginOrCreate(ctx, "bob")

	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	msvc.Record(ctx, alice.ID, "bob", "win", at) // alice 赢，bob 输

	view, err := msvc.ListByPlayer(ctx, bob.ID, 50, 0)
	if err != nil {
		t.Fatalf("ListByPlayer error: %v", err)
	}
	mv := view[0]
	if mv.Opponent != "alice" {
		t.Fatalf("opponent = %q, want alice", mv.Opponent)
	}
	if mv.Result != "loss" {
		t.Fatalf("result = %q, want loss", mv.Result)
	}
	if mv.Delta >= 0 {
		t.Fatalf("loser delta should be negative, got %d", mv.Delta)
	}
}

func TestMatchService_History_PrependsStartPoint(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	createdAt := alice.CreatedAt
	at := createdAt.Add(time.Hour)
	msvc.Record(ctx, alice.ID, "bob", "win", at)

	points, err := msvc.History(ctx, alice.ID, createdAt)
	if err != nil {
		t.Fatalf("History error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points (start + 1 match), got %d", len(points))
	}
	if !points[0].PlayedAt.Equal(createdAt.UTC()) {
		t.Fatalf("first point time = %v, want %v", points[0].PlayedAt, createdAt.UTC())
	}
	if points[0].Rating != domain.DefaultRating {
		t.Fatalf("first point rating = %d, want %d", points[0].Rating, domain.DefaultRating)
	}
	if points[1].Rating <= domain.DefaultRating {
		t.Fatalf("second point should reflect a win, got %d", points[1].Rating)
	}
}
```

- [ ] 运行确认失败：

```
cd server && go test ./internal/service/ -run TestMatchService
```

预期失败：编译错误 `undefined: NewMatchService`（以及 `RecordResult`、`MatchView`、`HistoryPoint` 未定义）。

- [ ] 写最小实现。创建 `server/internal/service/match.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

// RecordResult 是录入一局后返回给调用方的结果（相对提交者视角）。
type RecordResult struct {
	MatchID           int64
	WinnerDelta       int
	LoserDelta        int
	NewSelfRating     int
	NewOpponentRating int
}

// MatchView 是某个玩家视角下的一条对局展示数据。
type MatchView struct {
	ID           int64
	Opponent     string
	Result       string // 相对查询玩家："win" | "loss"
	RatingBefore int
	RatingAfter  int
	Delta        int
	PlayedAt     time.Time
}

// HistoryPoint 是历史曲线上的一个 (时间, 分数) 点。
type HistoryPoint struct {
	PlayedAt time.Time
	Rating   int
}

// MatchService 负责对局录入与查询。
type MatchService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewMatchService 构造 MatchService。
func NewMatchService(q *sqlc.Queries, db *sql.DB) *MatchService {
	return &MatchService{q: q, db: db}
}

// Record 录入一局对局。result="win" 表示提交者获胜（winner=submitter）。
// 整个读-算-写过程在一个事务内完成，对应 spec 的 BEGIN IMMEDIATE 语义。
func (s *MatchService) Record(ctx context.Context, submitterID int64, opponentUsername string, result string, playedAt time.Time) (RecordResult, error) {
	// 开一个可写事务。modernc sqlite 在事务内首次写时即获取写锁，等价于 BEGIN IMMEDIATE 的目的：
	// 避免两个并发录入读到同一份 rating 后互相覆盖。busy_timeout=5000 已在 db.New 设置。
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}
	defer func() { _ = tx.Rollback() }() // 提交成功后 Rollback 是 no-op

	qtx := s.q.WithTx(tx)

	// 查对手。
	opponent, err := qtx.GetPlayerByUsername(ctx, opponentUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecordResult{}, domain.ErrPlayerNotFound
		}
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}
	if opponent.ID == submitterID {
		return RecordResult{}, domain.ErrSelfMatch
	}

	// 查提交者。
	submitter, err := qtx.GetPlayerByID(ctx, submitterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecordResult{}, domain.ErrPlayerNotFound
		}
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	// 根据 result 决定谁是 winner。
	var winnerID, loserID int64
	var winnerBefore, loserBefore int
	switch result {
	case "win":
		winnerID, loserID = submitter.ID, opponent.ID
		winnerBefore, loserBefore = int(submitter.Rating), int(opponent.Rating)
	case "loss":
		winnerID, loserID = opponent.ID, submitter.ID
		winnerBefore, loserBefore = int(opponent.Rating), int(submitter.Rating)
	default:
		return RecordResult{}, domain.ErrInvalidParam
	}

	delta := domain.ComputeDelta(winnerBefore, loserBefore)
	winnerAfter := winnerBefore + delta
	loserAfter := loserBefore - delta

	now := time.Now().UTC()
	created, err := qtx.CreateMatch(ctx, sqlc.CreateMatchParams{
		WinnerID:           winnerID,
		LoserID:            loserID,
		SubmitterID:        submitterID,
		WinnerRatingBefore: int64(winnerBefore),
		LoserRatingBefore:  int64(loserBefore),
		WinnerRatingAfter:  int64(winnerAfter),
		LoserRatingAfter:   int64(loserAfter),
		WinnerDelta:        int64(delta),
		LoserDelta:         int64(-delta),
		PlayedAt:           formatTime(playedAt),
		CreatedAt:          formatTime(now),
	})
	if err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	if err := qtx.UpdatePlayerRating(ctx, sqlc.UpdatePlayerRatingParams{
		ID:     winnerID,
		Rating: int64(winnerAfter),
	}); err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}
	if err := qtx.UpdatePlayerRating(ctx, sqlc.UpdatePlayerRatingParams{
		ID:     loserID,
		Rating: int64(loserAfter),
	}); err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	if err := tx.Commit(); err != nil {
		return RecordResult{}, domain.ErrInternal.WithCause(err)
	}

	// 组装相对提交者视角的返回值。
	res := RecordResult{MatchID: created.ID}
	if result == "win" {
		res.WinnerDelta = delta
		res.LoserDelta = -delta
		res.NewSelfRating = winnerAfter
		res.NewOpponentRating = loserAfter
	} else {
		res.WinnerDelta = delta
		res.LoserDelta = -delta
		res.NewSelfRating = loserAfter   // 提交者是 loser
		res.NewOpponentRating = winnerAfter
	}
	return res, nil
}

// ListGlobal 返回全局对局流（不含已删除），按 played_at DESC。
// Opponent/Result 等视角字段相对 winner 渲染（全局流无"当前玩家"概念，统一以 winner 为主体）。
func (s *MatchService) ListGlobal(ctx context.Context, limit, offset int) ([]MatchView, error) {
	rows, err := s.q.ListGlobalMatches(ctx, sqlc.ListGlobalMatchesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	views := make([]MatchView, 0, len(rows))
	for _, m := range rows {
		// 全局流以 winner 为主体：Result 恒为 "win"，Opponent 为 loser。
		opp, err := s.usernameOf(ctx, m.LoserID)
		if err != nil {
			return nil, err
		}
		playedAt, perr := parseTime(m.PlayedAt)
		if perr != nil {
			return nil, domain.ErrInternal.WithCause(perr)
		}
		views = append(views, MatchView{
			ID:           m.ID,
			Opponent:     opp,
			Result:       "win",
			RatingBefore: int(m.WinnerRatingBefore),
			RatingAfter:  int(m.WinnerRatingAfter),
			Delta:        int(m.WinnerDelta),
			PlayedAt:     playedAt,
		})
	}
	return views, nil
}

// ListByPlayer 返回指定玩家的对局，所有字段相对该玩家渲染。
func (s *MatchService) ListByPlayer(ctx context.Context, playerID int64, limit, offset int) ([]MatchView, error) {
	rows, err := s.q.ListPlayerMatches(ctx, sqlc.ListPlayerMatchesParams{
		WinnerID: playerID,
		LoserID:  playerID,
		Limit:    int64(limit),
		Offset:   int64(offset),
	})
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	views := make([]MatchView, 0, len(rows))
	for _, m := range rows {
		playedAt, perr := parseTime(m.PlayedAt)
		if perr != nil {
			return nil, domain.ErrInternal.WithCause(perr)
		}
		mv := MatchView{ID: m.ID, PlayedAt: playedAt}
		if m.WinnerID == playerID {
			oppName, oerr := s.usernameOf(ctx, m.LoserID)
			if oerr != nil {
				return nil, oerr
			}
			mv.Opponent = oppName
			mv.Result = "win"
			mv.RatingBefore = int(m.WinnerRatingBefore)
			mv.RatingAfter = int(m.WinnerRatingAfter)
			mv.Delta = int(m.WinnerDelta)
		} else {
			oppName, oerr := s.usernameOf(ctx, m.WinnerID)
			if oerr != nil {
				return nil, oerr
			}
			mv.Opponent = oppName
			mv.Result = "loss"
			mv.RatingBefore = int(m.LoserRatingBefore)
			mv.RatingAfter = int(m.LoserRatingAfter)
			mv.Delta = int(m.LoserDelta)
		}
		views = append(views, mv)
	}
	return views, nil
}

// History 返回该玩家的历史曲线点，开头 prepend (createdAt, DefaultRating) 作为起点。
func (s *MatchService) History(ctx context.Context, playerID int64, createdAt time.Time) ([]HistoryPoint, error) {
	rows, err := s.q.GetPlayerHistory(ctx, sqlc.GetPlayerHistoryParams{
		WinnerID: playerID,
		LoserID:  playerID,
	})
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	points := make([]HistoryPoint, 0, len(rows)+1)
	points = append(points, HistoryPoint{PlayedAt: createdAt.UTC(), Rating: domain.DefaultRating})
	for _, r := range rows {
		playedAt, perr := parseTime(r.PlayedAt)
		if perr != nil {
			return nil, domain.ErrInternal.WithCause(perr)
		}
		var rating int
		if r.WinnerID == playerID {
			rating = int(r.WinnerRatingAfter)
		} else {
			rating = int(r.LoserRatingAfter)
		}
		points = append(points, HistoryPoint{PlayedAt: playedAt, Rating: rating})
	}
	return points, nil
}

// usernameOf 查某玩家的用户名（用于组装对局视角的对手名）。
func (s *MatchService) usernameOf(ctx context.Context, id int64) (string, error) {
	p, err := s.q.GetPlayerByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrPlayerNotFound
		}
		return "", domain.ErrInternal.WithCause(err)
	}
	return p.Username, nil
}
```

> 实现要点说明：
> - 事务：`db.BeginTx` 配合 `s.q.WithTx(tx)` 让所有读写走同一事务。`sql.LevelSerializable` 在 modernc sqlite 下映射到独占写语义；真正阻止并发覆盖的是 sqlite 的事务写锁 + `busy_timeout`（已在 `db.New` 设置）。`defer tx.Rollback()` 在成功 `Commit` 后是 no-op，失败路径自动回滚。
> - 零和：`loser_delta = -delta`，且 `winner_after = winner_before + delta`、`loser_after = loser_before - delta`，与迁移里三条 CHECK 一致；插入若违反 CHECK 会报错并回滚。
> - `GetPlayerHistoryRow` 字段为 `PlayedAt`、`WinnerID`、`LoserID`、`WinnerRatingAfter`、`LoserRatingAfter`（契约规定的投影列）。
> - `ListGlobalMatchesParams` / `ListPlayerMatchesParams` 的 `Limit`、`Offset` 为 `int64`。

- [ ] 运行确认通过：

```
cd server && go test ./internal/service/ -run TestMatchService -v
```

预期每个子测试 `--- PASS`，含 `TestMatchService_Record_SumConservedOver100Games` 通过（守恒）。

- [ ] 额外运行 race 检测（事务正确性）：

```
cd server && go test -race ./internal/service/ -run TestMatchService
```

预期 `ok`，无 data race 报告。

- [ ] commit：

```
git add server/internal/service/match.go server/internal/service/match_test.go
git commit -m "feat(service): implement MatchService (record in tx, views, history)"
```

---

### Task 5: `LeaderboardService`

实现 `List`（`ListPlayersByRating` + 对每人 `CountPlayerWinsLosses` 算 `GamesPlayed`/`WinRate`，用 `domain.Dan` 算段位，按 rating 序赋 `Rank`，`min_games` 用 `>=` 过滤）、`CompareData`（对每个用户名取 history 组装 `CompareSeries` 配 5 色板循环；`head_to_head` 生成 C(n,2) 对，`AWins`/`BWins` 仅计未删除局）。

**Files:**
- Create: `server/internal/service/leaderboard.go`
- Test: `server/internal/service/leaderboard_test.go`

步骤：

- [ ] 写失败测试。创建 `server/internal/service/leaderboard_test.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"testing"
	"time"

	"go_ultra/internal/domain"
)

func TestLeaderboardService_List_RankAndStats(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")

	// alice 赢一局 → alice 升、bob 降
	msvc.Record(ctx, alice.ID, "bob", "win", time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC))

	rows, err := lsvc.List(ctx, 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// rank 1 应为分高者 alice
	if rows[0].Rank != 1 || rows[0].Username != "alice" {
		t.Fatalf("rank 1 should be alice, got rank=%d user=%q", rows[0].Rank, rows[0].Username)
	}
	if rows[1].Rank != 2 || rows[1].Username != "bob" {
		t.Fatalf("rank 2 should be bob, got rank=%d user=%q", rows[1].Rank, rows[1].Username)
	}
	// alice：1 胜 0 负，games=1，winrate=1.0
	if rows[0].GamesPlayed != 1 || rows[0].WinRate != 1.0 {
		t.Fatalf("alice stats wrong: games=%d winrate=%v", rows[0].GamesPlayed, rows[0].WinRate)
	}
	// Dan 必须与 domain.Dan 一致
	if rows[0].Dan != domain.Dan(rows[0].Rating) {
		t.Fatalf("alice dan = %d, want %d", rows[0].Dan, domain.Dan(rows[0].Rating))
	}
}

func TestLeaderboardService_List_MinGamesFilter(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")
	psvc.LoginOrCreate(ctx, "carol") // 0 局

	msvc.Record(ctx, alice.ID, "bob", "win", time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC))

	// min_games=1：carol（0 局）必须被过滤掉
	rows, err := lsvc.List(ctx, 1)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	for _, r := range rows {
		if r.Username == "carol" {
			t.Fatalf("carol (0 games) should be filtered out by min_games=1")
		}
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows with min_games=1, got %d", len(rows))
	}
	// rank 在过滤后仍连续：1,2
	if rows[0].Rank != 1 || rows[1].Rank != 2 {
		t.Fatalf("ranks not contiguous after filter: %d, %d", rows[0].Rank, rows[1].Rank)
	}
}

func TestLeaderboardService_CompareData_SeriesAndHeadToHead(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	psvc.LoginOrCreate(ctx, "bob")
	psvc.LoginOrCreate(ctx, "carol")

	base := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	// alice vs bob：alice 赢 2, bob 赢 1
	msvc.Record(ctx, alice.ID, "bob", "win", base.Add(1*time.Minute))
	msvc.Record(ctx, alice.ID, "bob", "win", base.Add(2*time.Minute))
	msvc.Record(ctx, alice.ID, "bob", "loss", base.Add(3*time.Minute)) // bob 赢

	res, err := lsvc.CompareData(ctx, []string{"alice", "bob", "carol"})
	if err != nil {
		t.Fatalf("CompareData error: %v", err)
	}
	if len(res.Series) != 3 {
		t.Fatalf("expected 3 series, got %d", len(res.Series))
	}
	// 每条 series 开头都有起点（createdAt, DefaultRating）
	for _, s := range res.Series {
		if len(s.Points) == 0 {
			t.Fatalf("series %q has no points", s.Username)
		}
		if s.Points[0].Rating != domain.DefaultRating {
			t.Fatalf("series %q first point rating = %d, want %d", s.Username, s.Points[0].Rating, domain.DefaultRating)
		}
		if s.Color == "" {
			t.Fatalf("series %q has empty color", s.Username)
		}
	}
	// 5 色板：前 3 条颜色取调色板前 3 个
	wantPalette := []string{"#4a9eff", "#7fd6a3", "#8b5cf6"}
	for i, s := range res.Series {
		if s.Color != wantPalette[i] {
			t.Fatalf("series[%d] color = %q, want %q", i, s.Color, wantPalette[i])
		}
	}
	// head_to_head：C(3,2)=3 对
	if len(res.HeadToHead) != 3 {
		t.Fatalf("expected 3 head-to-head pairs, got %d", len(res.HeadToHead))
	}
	// 找 alice-bob 对：alice 2 胜 bob 1 胜
	var found bool
	for _, h := range res.HeadToHead {
		if (h.A == "alice" && h.B == "bob") || (h.A == "bob" && h.B == "alice") {
			found = true
			if h.A == "alice" {
				if h.AWins != 2 || h.BWins != 1 {
					t.Fatalf("alice-bob h2h = %d-%d, want 2-1", h.AWins, h.BWins)
				}
			} else { // A==bob
				if h.AWins != 1 || h.BWins != 2 {
					t.Fatalf("bob-alice h2h = %d-%d, want 1-2", h.AWins, h.BWins)
				}
			}
		}
	}
	if !found {
		t.Fatalf("alice-bob pair not found in head-to-head")
	}
}

func TestLeaderboardService_CompareData_ColorWrapsAfterFive(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	names := []string{"u1", "u2", "u3", "u4", "u5", "u6"}
	for _, n := range names {
		psvc.LoginOrCreate(ctx, n)
	}
	res, err := lsvc.CompareData(ctx, names)
	if err != nil {
		t.Fatalf("CompareData error: %v", err)
	}
	if len(res.Series) != 6 {
		t.Fatalf("expected 6 series, got %d", len(res.Series))
	}
	// 第 6 条（index 5）颜色应循环回调色板第 1 个
	if res.Series[5].Color != res.Series[0].Color {
		t.Fatalf("color did not wrap: series[5]=%q series[0]=%q", res.Series[5].Color, res.Series[0].Color)
	}
}

func TestLeaderboardService_CompareData_PlayerNotFound(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	lsvc := NewLeaderboardService(q, sqlDB)
	ctx := testCtx(t)

	psvc.LoginOrCreate(ctx, "alice")
	_, err := lsvc.CompareData(ctx, []string{"alice", "ghost"})
	if err == nil {
		t.Fatalf("expected error for unknown username, got nil")
	}
}
```

- [ ] 运行确认失败：

```
cd server && go test ./internal/service/ -run TestLeaderboardService
```

预期失败：编译错误 `undefined: NewLeaderboardService`（及 `LeaderboardRow`、`CompareSeries`、`HeadToHead`、`CompareResult` 未定义）。

- [ ] 写最小实现。创建 `server/internal/service/leaderboard.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"context"
	"database/sql"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
)

// comparePalette 是对比图的 5 色板（与契约/前端 ECharts 5 色板一致）。
var comparePalette = []string{"#4a9eff", "#7fd6a3", "#8b5cf6", "#e0c47d", "#f08080"}

// LeaderboardRow 是排行榜的一行。
type LeaderboardRow struct {
	Rank        int
	Username    string
	Rating      int
	Dan         int
	GamesPlayed int
	WinRate     float64
}

// CompareSeries 是对比图中某玩家的一条曲线。
type CompareSeries struct {
	Username string
	Color    string
	Points   []HistoryPoint
}

// HeadToHead 是两名玩家的交手统计。
type HeadToHead struct {
	A     string
	B     string
	AWins int
	BWins int
}

// CompareResult 是 /compare 的完整结果。
type CompareResult struct {
	Series     []CompareSeries
	HeadToHead []HeadToHead
}

// LeaderboardService 负责排行榜与多人对比。
type LeaderboardService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewLeaderboardService 构造 LeaderboardService。
func NewLeaderboardService(q *sqlc.Queries, db *sql.DB) *LeaderboardService {
	return &LeaderboardService{q: q, db: db}
}

// List 返回排行榜。按 rating 降序赋连续 rank；min_games 用 >= 过滤（games < minGames 的玩家不出现）。
func (s *LeaderboardService) List(ctx context.Context, minGames int) ([]LeaderboardRow, error) {
	players, err := s.q.ListPlayersByRating(ctx)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	rows := make([]LeaderboardRow, 0, len(players))
	rank := 0
	for _, p := range players {
		counts, cerr := s.q.CountPlayerWinsLosses(ctx, p.ID)
		if cerr != nil {
			return nil, domain.ErrInternal.WithCause(cerr)
		}
		wins := int(counts.Wins)
		losses := int(counts.Losses)
		games := wins + losses
		if games < minGames {
			continue
		}
		var winRate float64
		if games > 0 {
			winRate = float64(wins) / float64(games)
		}
		rank++
		rows = append(rows, LeaderboardRow{
			Rank:        rank,
			Username:    p.Username,
			Rating:      int(p.Rating),
			Dan:         domain.Dan(int(p.Rating)),
			GamesPlayed: games,
			WinRate:     winRate,
		})
	}
	return rows, nil
}

// CompareData 为每个用户名组装一条历史曲线（配 5 色板循环），并生成所有 C(n,2) 对的交手统计。
func (s *LeaderboardService) CompareData(ctx context.Context, usernames []string) (CompareResult, error) {
	type entry struct {
		player domain.Player
		points []HistoryPoint
	}
	entries := make([]entry, 0, len(usernames))
	idByName := make(map[string]int64, len(usernames))

	matchSvc := NewMatchService(s.q, s.db)

	for i, name := range usernames {
		row, err := s.q.GetPlayerByUsername(ctx, name)
		if err != nil {
			if err == sql.ErrNoRows {
				return CompareResult{}, domain.ErrPlayerNotFound
			}
			return CompareResult{}, domain.ErrInternal.WithCause(err)
		}
		p, cerr := toDomainPlayer(row)
		if cerr != nil {
			return CompareResult{}, domain.ErrInternal.WithCause(cerr)
		}
		points, herr := matchSvc.History(ctx, p.ID, p.CreatedAt)
		if herr != nil {
			return CompareResult{}, herr
		}
		entries = append(entries, entry{player: p, points: points})
		idByName[p.Username] = p.ID
		_ = i
	}

	series := make([]CompareSeries, 0, len(entries))
	for i, e := range entries {
		series = append(series, CompareSeries{
			Username: e.player.Username,
			Color:    comparePalette[i%len(comparePalette)],
			Points:   e.points,
		})
	}

	// 生成 C(n,2) 对，AWins/BWins 仅计未删除局。
	heads := make([]HeadToHead, 0)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			aName := entries[i].player.Username
			bName := entries[j].player.Username
			aID := entries[i].player.ID
			bID := entries[j].player.ID
			aWins, bWins, err := s.headToHead(ctx, aID, bID)
			if err != nil {
				return CompareResult{}, err
			}
			heads = append(heads, HeadToHead{A: aName, B: bName, AWins: aWins, BWins: bWins})
		}
	}

	return CompareResult{Series: series, HeadToHead: heads}, nil
}

// headToHead 统计 a 与 b 之间未删除对局里 a 的胜场与 b 的胜场。
func (s *LeaderboardService) headToHead(ctx context.Context, aID, bID int64) (aWins, bWins int, err error) {
	// 复用 ListPlayerMatches 取 a 的全部未删除对局，再筛对手为 b 的。
	rows, qerr := s.q.ListPlayerMatches(ctx, sqlc.ListPlayerMatchesParams{
		WinnerID: aID,
		LoserID:  aID,
		Limit:    1000000,
		Offset:   0,
	})
	if qerr != nil {
		return 0, 0, domain.ErrInternal.WithCause(qerr)
	}
	for _, m := range rows {
		// 仅统计 a 与 b 之间的对局。
		isPair := (m.WinnerID == aID && m.LoserID == bID) || (m.WinnerID == bID && m.LoserID == aID)
		if !isPair {
			continue
		}
		if m.WinnerID == aID {
			aWins++
		} else {
			bWins++
		}
	}
	return aWins, bWins, nil
}
```

> 实现要点说明：
> - `ListPlayerMatches` 已过滤 `deleted_at IS NULL`，所以 `headToHead` 天然"仅计未删除局"。
> - `min_games` 过滤后 `rank` 仍连续：rank 在通过过滤的行上递增。
> - 颜色循环：`comparePalette[i % len]`，第 6 条回到第 1 个。
> - `CompareData` 依赖 `MatchService.History` 复用同一份起点 prepend 逻辑，避免重复实现。

- [ ] 运行确认通过：

```
cd server && go test ./internal/service/ -run TestLeaderboardService -v
```

预期每个子测试 `--- PASS`。

- [ ] commit：

```
git add server/internal/service/leaderboard.go server/internal/service/leaderboard_test.go
git commit -m "feat(service): implement LeaderboardService (ranking, compare series, head-to-head)"
```

---

### Task 6: `AdminService` 与软删除/恢复联动集成测试

实现 `EnsurePassword`（无 `admin_password_hash` 则生成 16 位随机密码 + bcrypt 存，返回明文 + `generated=true`；已存在返回 `("", false, nil)`，二次调用 `generated=false`）、`VerifyPassword`、`CreateAdminSession`（`session.NewToken` + 30 分钟过期，写 `admin_sessions`）、`CheckAdminSession`、`SoftDelete`（`deleted_by=NULL`）、`Restore`。本任务的集成测试同时覆盖与 `MatchService` 的联动：软删除后 `ListByPlayer`/`ListGlobal` 不返回该局、`ListDeletedMatches` 返回；`Restore` 后重新出现。

**Files:**
- Create: `server/internal/service/admin.go`
- Test: `server/internal/service/admin_test.go`

步骤：

- [ ] 写失败测试。创建 `server/internal/service/admin_test.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"testing"
	"time"
)

func TestAdminService_EnsurePassword_GenerateThenIdempotent(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	pw, generated, err := svc.EnsurePassword(ctx)
	if err != nil {
		t.Fatalf("EnsurePassword error: %v", err)
	}
	if !generated {
		t.Fatalf("first EnsurePassword should report generated=true")
	}
	if len(pw) != 16 {
		t.Fatalf("generated password length = %d, want 16", len(pw))
	}
	// 生成的明文必须能被 VerifyPassword 接受
	ok, err := svc.VerifyPassword(ctx, pw)
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !ok {
		t.Fatalf("generated password failed verification")
	}

	// 二次调用：generated=false，明文为空
	pw2, generated2, err := svc.EnsurePassword(ctx)
	if err != nil {
		t.Fatalf("second EnsurePassword error: %v", err)
	}
	if generated2 {
		t.Fatalf("second EnsurePassword should report generated=false")
	}
	if pw2 != "" {
		t.Fatalf("second EnsurePassword should return empty plaintext, got %q", pw2)
	}
	// 原密码仍然有效
	ok, _ = svc.VerifyPassword(ctx, pw)
	if !ok {
		t.Fatalf("original password should still verify after second EnsurePassword")
	}
}

func TestAdminService_VerifyPassword_WrongPassword(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	if _, _, err := svc.EnsurePassword(ctx); err != nil {
		t.Fatalf("EnsurePassword error: %v", err)
	}
	ok, err := svc.VerifyPassword(ctx, "definitely-wrong")
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if ok {
		t.Fatalf("wrong password should not verify")
	}
}

func TestAdminService_Session_CreateAndCheck(t *testing.T) {
	sqlDB, q := newTestDB(t)
	svc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	token, expiresAt, err := svc.CreateAdminSession(ctx)
	if err != nil {
		t.Fatalf("CreateAdminSession error: %v", err)
	}
	if token == "" {
		t.Fatalf("empty token")
	}
	// 过期时间应在约 30 分钟后（容差 1 分钟）
	wantMin := time.Now().UTC().Add(29 * time.Minute)
	wantMax := time.Now().UTC().Add(31 * time.Minute)
	if expiresAt.Before(wantMin) || expiresAt.After(wantMax) {
		t.Fatalf("expires_at out of range: %v", expiresAt)
	}

	ok, exp, err := svc.CheckAdminSession(ctx, token)
	if err != nil {
		t.Fatalf("CheckAdminSession error: %v", err)
	}
	if !ok {
		t.Fatalf("valid session should check ok")
	}
	if !exp.Equal(expiresAt) {
		t.Fatalf("checked expires_at = %v, want %v", exp, expiresAt)
	}

	// 不存在的 token
	ok, _, err = svc.CheckAdminSession(ctx, "no-such-token")
	if err != nil {
		t.Fatalf("CheckAdminSession(unknown) error: %v", err)
	}
	if ok {
		t.Fatalf("unknown token should not check ok")
	}
}

func TestAdminService_SoftDelete_HidesFromQueries_RestoreBringsBack(t *testing.T) {
	sqlDB, q := newTestDB(t)
	psvc := NewPlayerService(q, sqlDB)
	msvc := NewMatchService(q, sqlDB)
	asvc := NewAdminService(q, sqlDB)
	ctx := testCtx(t)

	alice, _ := psvc.LoginOrCreate(ctx, "alice")
	bob, _ := psvc.LoginOrCreate(ctx, "bob")

	at := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	res, err := msvc.Record(ctx, alice.ID, "bob", "win", at)
	if err != nil {
		t.Fatalf("Record error: %v", err)
	}

	// 删除前：全局/各玩家视角都能看到
	if g, _ := msvc.ListGlobal(ctx, 50, 0); len(g) != 1 {
		t.Fatalf("before delete: ListGlobal expected 1, got %d", len(g))
	}
	if pv, _ := msvc.ListByPlayer(ctx, alice.ID, 50, 0); len(pv) != 1 {
		t.Fatalf("before delete: alice ListByPlayer expected 1, got %d", len(pv))
	}
	if dv, _ := asvc.ListDeleted(ctx); len(dv) != 0 {
		t.Fatalf("before delete: ListDeleted expected 0, got %d", len(dv))
	}

	// 软删除
	if err := asvc.SoftDelete(ctx, res.MatchID); err != nil {
		t.Fatalf("SoftDelete error: %v", err)
	}

	// 删除后：普通查询不返回，已删除列表返回
	if g, _ := msvc.ListGlobal(ctx, 50, 0); len(g) != 0 {
		t.Fatalf("after delete: ListGlobal expected 0, got %d", len(g))
	}
	if pv, _ := msvc.ListByPlayer(ctx, alice.ID, 50, 0); len(pv) != 0 {
		t.Fatalf("after delete: alice ListByPlayer expected 0, got %d", len(pv))
	}
	if pv, _ := msvc.ListByPlayer(ctx, bob.ID, 50, 0); len(pv) != 0 {
		t.Fatalf("after delete: bob ListByPlayer expected 0, got %d", len(pv))
	}
	deleted, err := asvc.ListDeleted(ctx)
	if err != nil {
		t.Fatalf("ListDeleted error: %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("after delete: ListDeleted expected 1, got %d", len(deleted))
	}
	if deleted[0].ID != res.MatchID {
		t.Fatalf("deleted match id = %d, want %d", deleted[0].ID, res.MatchID)
	}
	// deleted_by 应为 NULL（管理员非 player）
	if deleted[0].DeletedBy != nil {
		t.Fatalf("deleted_by should be nil, got %v", *deleted[0].DeletedBy)
	}

	// 恢复
	if err := asvc.Restore(ctx, res.MatchID); err != nil {
		t.Fatalf("Restore error: %v", err)
	}
	if g, _ := msvc.ListGlobal(ctx, 50, 0); len(g) != 1 {
		t.Fatalf("after restore: ListGlobal expected 1, got %d", len(g))
	}
	if dv, _ := asvc.ListDeleted(ctx); len(dv) != 0 {
		t.Fatalf("after restore: ListDeleted expected 0, got %d", len(dv))
	}
}
```

- [ ] 运行确认失败：

```
cd server && go test ./internal/service/ -run TestAdminService
```

预期失败：编译错误 `undefined: NewAdminService`（及 `ListDeleted` 方法未定义）。

- [ ] 写最小实现。创建 `server/internal/service/admin.go`，内容如下（完整可粘贴）：

```go
package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/domain"
	"go_ultra/internal/session"
)

const adminPasswordHashKey = "admin_password_hash"

// AdminService 负责管理员密码、会话与对局软删除/恢复。
type AdminService struct {
	q  *sqlc.Queries
	db *sql.DB
}

// NewAdminService 构造 AdminService。
func NewAdminService(q *sqlc.Queries, db *sql.DB) *AdminService {
	return &AdminService{q: q, db: db}
}

// EnsurePassword 确保存在管理员密码。
// 若 settings 无 admin_password_hash：用 GenerateAdminPassword 生成可读明文，bcrypt 后存入，返回 (明文, true, nil)。
// 已存在：返回 ("", false, nil)。
func (s *AdminService) EnsurePassword(ctx context.Context) (string, bool, error) {
	_, err := s.q.GetSetting(ctx, adminPasswordHashKey)
	if err == nil {
		return "", false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, domain.ErrInternal.WithCause(err)
	}

	plaintext, hash, gerr := GenerateAdminPassword()
	if gerr != nil {
		return "", false, domain.ErrInternal.WithCause(gerr)
	}
	if serr := s.q.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   adminPasswordHashKey,
		Value: hash,
	}); serr != nil {
		return "", false, domain.ErrInternal.WithCause(serr)
	}
	return plaintext, true, nil
}

// VerifyPassword 校验明文密码是否匹配存储的 bcrypt 哈希。
func (s *AdminService) VerifyPassword(ctx context.Context, pw string) (bool, error) {
	hash, err := s.q.GetSetting(ctx, adminPasswordHashKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, domain.ErrInternal.WithCause(err)
	}
	if cmpErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); cmpErr != nil {
		return false, nil
	}
	return true, nil
}

// CreateAdminSession 生成一个 30 分钟有效的管理员会话并落库。
func (s *AdminService) CreateAdminSession(ctx context.Context) (string, time.Time, error) {
	token, err := session.NewToken()
	if err != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(err)
	}
	now := time.Now().UTC()
	expiresAt := now.Add(session.AdminSessionTTL)
	if serr := s.q.CreateAdminSession(ctx, sqlc.CreateAdminSessionParams{
		Token:     token,
		CreatedAt: formatTime(now),
		ExpiresAt: formatTime(expiresAt),
	}); serr != nil {
		return "", time.Time{}, domain.ErrInternal.WithCause(serr)
	}
	return token, expiresAt, nil
}

// CheckAdminSession 校验 token 对应的会话是否存在且未过期。
func (s *AdminService) CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error) {
	row, err := s.q.GetAdminSession(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, domain.ErrInternal.WithCause(err)
	}
	expiresAt, perr := parseTime(row.ExpiresAt)
	if perr != nil {
		return false, time.Time{}, domain.ErrInternal.WithCause(perr)
	}
	if time.Now().UTC().After(expiresAt) {
		return false, expiresAt, nil
	}
	return true, expiresAt, nil
}

// SoftDelete 软删除一局对局，deleted_by 置 NULL（管理员非 player）。
func (s *AdminService) SoftDelete(ctx context.Context, matchID int64) error {
	if err := s.q.SoftDeleteMatch(ctx, sqlc.SoftDeleteMatchParams{
		DeletedAt: sql.NullString{String: formatTime(time.Now().UTC()), Valid: true},
		DeletedBy: sql.NullInt64{Valid: false},
		ID:        matchID,
	}); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}

// Restore 取消一局对局的软删除（幂等）。
func (s *AdminService) Restore(ctx context.Context, matchID int64) error {
	if err := s.q.RestoreMatch(ctx, matchID); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}

// ListDeleted 返回所有已软删除的对局，按 deleted_at DESC。
// （底层 sqlc 查询名为 ListDeletedMatches；service 方法名按 http 层接口约定为 ListDeleted。）
func (s *AdminService) ListDeleted(ctx context.Context) ([]domain.Match, error) {
	rows, err := s.q.ListDeletedMatches(ctx)
	if err != nil {
		return nil, domain.ErrInternal.WithCause(err)
	}
	matches := make([]domain.Match, 0, len(rows))
	for _, r := range rows {
		m, cerr := toDomainMatch(r)
		if cerr != nil {
			return nil, domain.ErrInternal.WithCause(cerr)
		}
		matches = append(matches, m)
	}
	return matches, nil
}

// DeleteAdminSession 删除指定管理员会话 token（登出）。幂等。
func (s *AdminService) DeleteAdminSession(ctx context.Context, token string) error {
	if err := s.q.DeleteAdminSession(ctx, token); err != nil {
		return domain.ErrInternal.WithCause(err)
	}
	return nil
}

// adminPasswordAlphabet 是可读、可输入的密码字符集（去除易混字符 0/O/1/l/I）。
const adminPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// GenerateAdminPassword 生成 16 位可读随机明文密码及其 bcrypt 哈希。
// 供 EnsurePassword 与 ResetPassword（阶段 7）共用，保证两条路径产出格式一致、字符可输入。
func GenerateAdminPassword() (plaintext, hash string, err error) {
	const n = 16
	buf := make([]byte, n)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = adminPasswordAlphabet[int(b)%len(adminPasswordAlphabet)]
	}
	plaintext = string(out)
	h, herr := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if herr != nil {
		return "", "", herr
	}
	return plaintext, string(h), nil
}
```

> 实现要点说明：
> - `EnsurePassword` 用 `GetSetting` 命中与否区分"已存在/需生成"；`sql.ErrNoRows` 表示需生成。`SetSetting` 是 UPSERT（契约规定），但这里只在不存在时写，幂等性由"先查后写"保证。
> - 密码生成统一走 `GenerateAdminPassword`（16 位，字符集去除易混字符），`EnsurePassword`（首启）与阶段 7 的 `ResetPassword`（重置）共用同一函数，保证明文格式一致且可被用户正常输入。
> - `ListDeleted` 是 service 方法名（http 层 `adminMatchSvc` 接口要求），底层调用 sqlc 查询 `ListDeletedMatches`。
> - `DeleteAdminSession` 包装 sqlc `DeleteAdminSession`，管理员登出用，幂等。
> - `SoftDeleteMatch` 的 `DeletedBy` 传 `sql.NullInt64{Valid:false}`（即 NULL），符合契约"deleted_by = NULL（管理员非 player）"。
> - `SoftDeleteMatchParams` 字段顺序按契约 `(deleted_at, deleted_by, id)`；sqlc 生成对应大驼峰字段 `DeletedAt`/`DeletedBy`/`ID`。
> - `Restore` 调用 `RestoreMatch(ctx, matchID)`（单参数，契约：`SET deleted_at=NULL, deleted_by=NULL WHERE id=?`），重复调用幂等。
> - `admin.go` 顶部 import 需含 `"crypto/rand"`、`"golang.org/x/crypto/bcrypt"`、`"go_ultra/internal/session"`（其余 `context`/`database/sql`/`errors`/`time`/`sqlc`/`domain` 已引入）。

- [ ] 运行确认通过：

```
cd server && go test ./internal/service/ -run TestAdminService -v
```

预期每个子测试 `--- PASS`，含软删除/恢复联动全部通过。

- [ ] commit：

```
git add server/internal/service/admin.go server/internal/service/admin_test.go
git commit -m "feat(service): implement AdminService (password, sessions, soft-delete/restore)"
```

---

### Task 7: 全量回归与覆盖率验证

确认整个 service 包所有集成测试在一起跑通、无 race、覆盖率达到 spec §9.4 的 `service/* ≥ 80%` 目标。

**Files:**
- Create:（无）
- Test:（运行已有全部测试）

步骤：

- [ ] 全量运行（含 verbose），确认无失败：

```
cd server && go test ./internal/service/ -v
```

预期所有 `--- PASS`，结尾 `ok  	go_ultra/internal/service`。

- [ ] race 检测全量运行（验证 `MatchService.Record` 的事务在并发下无数据竞争）：

```
cd server && go test -race ./internal/service/
```

预期 `ok`，无 `WARNING: DATA RACE`。

- [ ] 覆盖率检查（必须 ≥ 80%）：

```
cd server && go test -cover ./internal/service/
```

预期输出形如 `ok  	go_ultra/internal/service	X.XXXs	coverage: 8X.X% of statements`，coverage ≥ 80.0%。若不足 80%，用下一条命令定位未覆盖分支并补测试，直至达标：

```
cd server && go test -coverprofile=coverage.out ./internal/service/ && go tool cover -func=coverage.out
```

逐函数查看覆盖率；对 `< 80%` 的函数补充针对其未覆盖分支的测试用例（例如错误路径），再回到本任务第一步重跑，直到 `-cover` 报告 ≥ 80%。完成后删除临时文件：

```
cd server && rm -f coverage.out
```

- [ ] commit（如本任务为达标补充了测试用例则提交；若无新增改动则跳过此步）：

```
git add server/internal/service/
git commit -m "test(service): raise integration coverage to >=80%"
```

---

**阶段 3 完成判据**：`server/internal/service` 下 `convert.go`、`player.go`、`match.go`、`leaderboard.go`、`admin.go` 五个生产文件 + 对应集成测试全部存在；`go test ./internal/service/` 通过；`go test -race ./internal/service/` 无竞争；`go test -cover ./internal/service/` ≥ 80%。四个服务的构造函数与方法签名严格遵循契约（`NewXxxService(q *sqlc.Queries, db *sql.DB)`，返回类型 `RecordResult`/`MatchView`/`HistoryPoint`/`LeaderboardRow`/`CompareSeries`/`HeadToHead`/`CompareResult`），未自创或改名任何 sqlc 查询、domain 类型或函数。

---

## 阶段 4: http 层` 标题开始，严格遵守任务撰写规则。

---

## 阶段 4: http 层 + 程序装配

本阶段实现 HTTP 层（session、middleware、handler、router）与程序装配（main.go）。前置假设：阶段 1（domain）、阶段 2（db + sqlc）、阶段 3（service）均已按契约实现完毕，可直接 import 使用。本阶段所有路径以仓库根 `go_ultra/` 为基准。

各 Task 内的命令默认在 `server/` 目录下执行（即 `go.mod` 所在目录）。每个 Task 末尾都有 commit 步骤。

---

### Task 1: session 层 —— Token 生成、TTL 常量、Cookie 名常量

**Files:**
- Create: `server/internal/session/session.go`
- Test: `server/internal/session/session_test.go`

本 Task 实现 `NewToken()`（`crypto/rand` 32 字节 → `base64.RawURLEncoding`）、会话 TTL 常量、Cookie 名常量。

步骤：

- [ ] 写失败测试 `server/internal/session/session_test.go`，内容完整如下：

```go
package session

import (
	"testing"
	"time"
)

func TestNewToken_Length(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken returned error: %v", err)
	}
	// 32 bytes raw -> base64.RawURLEncoding 无填充，长度 = ceil(32*8/6) = 43
	if len(tok) != 43 {
		t.Fatalf("token length = %d, want 43; token=%q", len(tok), tok)
	}
}

func TestNewToken_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		tok, err := NewToken()
		if err != nil {
			t.Fatalf("NewToken returned error at i=%d: %v", i, err)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("duplicate token generated at i=%d: %q", i, tok)
		}
		seen[tok] = struct{}{}
	}
}

func TestNewToken_URLSafe(t *testing.T) {
	tok, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken returned error: %v", err)
	}
	for _, r := range tok {
		ok := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			t.Fatalf("token contains non-url-safe char %q in %q", r, tok)
		}
	}
}

func TestConstants(t *testing.T) {
	if PlayerSessionTTL != 30*24*time.Hour {
		t.Fatalf("PlayerSessionTTL = %v, want 720h", PlayerSessionTTL)
	}
	if AdminSessionTTL != 30*time.Minute {
		t.Fatalf("AdminSessionTTL = %v, want 30m", AdminSessionTTL)
	}
	if PlayerCookieName != "go_ultra_session" {
		t.Fatalf("PlayerCookieName = %q, want go_ultra_session", PlayerCookieName)
	}
	if AdminCookieName != "go_ultra_admin" {
		t.Fatalf("AdminCookieName = %q, want go_ultra_admin", AdminCookieName)
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/session/
```

预期失败（编译错误，因为 `session.go` 尚未提供这些符号），输出类似：

```
# go_ultra/internal/session [go_ultra/internal/session.test]
./session_test.go:11:18: undefined: NewToken
./session_test.go:...: undefined: PlayerSessionTTL
FAIL	go_ultra/internal/session [build failed]
```

- [ ] 写最小实现 `server/internal/session/session.go`，内容完整如下：

```go
package session

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Cookie 名常量。
const (
	PlayerCookieName = "go_ultra_session"
	AdminCookieName  = "go_ultra_admin"
)

// 会话有效期常量。
const (
	PlayerSessionTTL = 30 * 24 * time.Hour // 30 天，滑动续期
	AdminSessionTTL  = 30 * time.Minute
)

// NewToken 生成一个 32 字节的密码学随机 token，
// 以无填充的 base64 URL 安全编码返回（长度固定 43）。
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
```

- [ ] 运行确认通过：

```
go test ./internal/session/
```

预期输出：

```
ok  	go_ultra/internal/session	0.0xxs
```

- [ ] commit：

```
git add server/internal/session/session.go server/internal/session/session_test.go
git commit -m "feat(session): add NewToken, TTL and cookie name constants"
```

---

### Task 2: 统一错误响应 helper —— respondError

**Files:**
- Create: `server/internal/handler/response.go`
- Test: `server/internal/handler/response_test.go`

本 Task 实现 handler 层统一错误响应 helper：从 `*domain.Error` 取 `Status/Code/Message`，输出 `{"error":{"code","message"}}`；非 `domain.Error` 一律映射为 `ErrInternal`（500）并用 zerolog 记录原始 `Cause`。后续 handler 全部依赖该 helper。

步骤：

- [ ] 写失败测试 `server/internal/handler/response_test.go`，内容完整如下：

```go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestRespondError_DomainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	respondError(c, domain.ErrPlayerNotFound)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var body errBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "PLAYER_NOT_FOUND" {
		t.Fatalf("code = %q, want PLAYER_NOT_FOUND", body.Error.Code)
	}
	if body.Error.Message != "玩家不存在" {
		t.Fatalf("message = %q, want 玩家不存在", body.Error.Message)
	}
}

func TestRespondError_DomainErrorWithCause(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	wrapped := domain.ErrInvalidParam.WithCause(errors.New("bad parse"))
	respondError(c, wrapped)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var body errBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "INVALID_PARAM" {
		t.Fatalf("code = %q, want INVALID_PARAM", body.Error.Code)
	}
}

func TestRespondError_NonDomainError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	respondError(c, errors.New("some random failure"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body errBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "INTERNAL" {
		t.Fatalf("code = %q, want INTERNAL", body.Error.Code)
	}
	if body.Error.Message != "服务器内部错误" {
		t.Fatalf("message = %q, want 服务器内部错误", body.Error.Message)
	}
}

func TestRespondError_NilError(t *testing.T) {
	// nil 也按 500 兜底处理，避免空指针。
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(ctxLogger, zerolog.Nop())

	respondError(c, nil)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/handler/
```

预期失败（编译错误，`respondError`、`ctxLogger`、`errBody` 涉及的符号未定义）：

```
# go_ultra/internal/handler [go_ultra/internal/handler.test]
./response_test.go:...: undefined: ctxLogger
./response_test.go:...: undefined: respondError
FAIL	go_ultra/internal/handler [build failed]
```

- [ ] 写最小实现 `server/internal/handler/response.go`，内容完整如下：

```go
package handler

import (
	"net/http"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// gin.Context 中存放共享值的 key 常量。
const (
	ctxLogger    = "logger"    // zerolog.Logger
	ctxRequestID = "requestID" // string
	ctxPlayerID  = "playerID"  // int64
)

// errorEnvelope 是统一错误响应的 JSON 形状：{"error":{"code","message"}}。
type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// loggerFrom 从 gin.Context 取出请求级 logger；不存在则返回 Nop。
func loggerFrom(c *gin.Context) zerolog.Logger {
	if v, ok := c.Get(ctxLogger); ok {
		if lg, ok := v.(zerolog.Logger); ok {
			return lg
		}
	}
	return zerolog.Nop()
}

// respondError 把任意 error 转换为统一错误响应并终止请求链。
// *domain.Error 直接取 Status/Code/Message；其它 error 一律 500 INTERNAL，
// 并把原始 error 作为 Cause 记入日志。
func respondError(c *gin.Context, err error) {
	var de *domain.Error
	if e, ok := err.(*domain.Error); ok && e != nil {
		de = e
	} else {
		de = domain.ErrInternal.WithCause(err)
	}

	if de.Status >= http.StatusInternalServerError {
		ev := loggerFrom(c).Error().
			Str("code", de.Code).
			Str("message", de.Message)
		if de.Cause != nil {
			ev = ev.Err(de.Cause)
		}
		ev.Msg("request failed")
	}

	c.AbortWithStatusJSON(de.Status, errorEnvelope{
		Error: errorPayload{Code: de.Code, Message: de.Message},
	})
}
```

- [ ] 运行确认通过：

```
go test ./internal/handler/
```

预期输出：

```
ok  	go_ultra/internal/handler	0.0xxs
```

- [ ] commit：

```
git add server/internal/handler/response.go server/internal/handler/response_test.go
git commit -m "feat(handler): add unified respondError helper and context keys"
```

---

### Task 3: middleware —— RequestID、Logger、Recover

**Files:**
- Create: `server/internal/middleware/middleware.go`
- Test: `server/internal/middleware/middleware_test.go`

本 Task 实现三个基础中间件：`RequestID()`（用 `google/uuid`，注入到 ctx 与响应头 `X-Request-ID`）、`Logger(zerolog)`（每请求一行结构化日志，并把请求级 logger 注入 ctx）、`Recover()`（panic → 500 统一 JSON）。

> 说明：middleware 与 handler 共用同一套 ctx key 字符串（`logger` / `requestID` / `playerID`）。为避免跨包循环依赖，middleware 包内自带这组常量定义（值与 handler 包一致，靠测试保证一致性）。

步骤：

- [ ] 先确认依赖已在 `go.mod`（阶段 2/3 可能已引入；若无则添加）。运行：

```
go get github.com/google/uuid@latest
go get github.com/rs/zerolog@latest
go get github.com/gin-gonic/gin@v1.10.0
```

预期输出包含 `go: added github.com/google/uuid ...`（已存在则无新增，正常）。

- [ ] 写失败测试 `server/internal/middleware/middleware_test.go`，内容完整如下：

```go
package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_SetsHeaderAndContext(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	var gotFromCtx string
	r.GET("/", func(c *gin.Context) {
		v, _ := c.Get(CtxRequestID)
		gotFromCtx, _ = v.(string)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	hdr := w.Header().Get("X-Request-ID")
	if hdr == "" {
		t.Fatalf("X-Request-ID header is empty")
	}
	if gotFromCtx == "" {
		t.Fatalf("request id not stored in context")
	}
	if hdr != gotFromCtx {
		t.Fatalf("header %q != context %q", hdr, gotFromCtx)
	}
}

func TestLogger_InjectsLoggerIntoContext(t *testing.T) {
	r := gin.New()
	base := zerolog.Nop()
	r.Use(RequestID())
	r.Use(Logger(base))
	found := false
	r.GET("/", func(c *gin.Context) {
		if _, ok := c.Get(CtxLogger); ok {
			found = true
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if !found {
		t.Fatalf("logger not injected into context")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestRecover_PanicBecomes500JSON(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger(zerolog.Nop()))
	r.Use(Recover())
	r.GET("/boom", func(c *gin.Context) {
		panic("kaboom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "INTERNAL" {
		t.Fatalf("code = %q, want INTERNAL", body.Error.Code)
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/middleware/
```

预期失败（编译错误，`RequestID`/`Logger`/`Recover`/`CtxRequestID`/`CtxLogger` 未定义）：

```
# go_ultra/internal/middleware [go_ultra/internal/middleware.test]
./middleware_test.go:...: undefined: RequestID
./middleware_test.go:...: undefined: CtxRequestID
FAIL	go_ultra/internal/middleware [build failed]
```

- [ ] 写最小实现 `server/internal/middleware/middleware.go`，内容完整如下：

```go
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// 与 handler 包共用的 gin.Context key（值必须一致）。
const (
	CtxLogger    = "logger"    // zerolog.Logger
	CtxRequestID = "requestID" // string
	CtxPlayerID  = "playerID"  // int64
)

// RequestID 为每个请求生成一个 UUID，存入 context 并写入响应头 X-Request-ID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		c.Set(CtxRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger 注入一个带 request_id 字段的请求级 logger，并在请求结束后输出一行访问日志。
func Logger(base zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID, _ := c.Get(CtxRequestID)
		ridStr, _ := reqID.(string)

		reqLogger := base.With().Str("request_id", ridStr).Logger()
		c.Set(CtxLogger, reqLogger)

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		var playerID int64
		if v, ok := c.Get(CtxPlayerID); ok {
			if pid, ok := v.(int64); ok {
				playerID = pid
			}
		}

		ev := reqLogger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int64("latency_ms", latency.Milliseconds())
		if playerID != 0 {
			ev = ev.Int64("player_id", playerID)
		}
		if len(c.Errors) > 0 {
			ev = ev.Str("error", c.Errors.String())
		}
		ev.Msg("http request")
	}
}

// Recover 捕获 handler 中的 panic，返回统一的 500 JSON，并记录堆栈信息。
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if v, ok := c.Get(CtxLogger); ok {
					if lg, ok := v.(zerolog.Logger); ok {
						lg.Error().Interface("panic", rec).Msg("recovered from panic")
					}
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL",
						"message": "服务器内部错误",
					},
				})
			}
		}()
		c.Next()
	}
}
```

- [ ] 运行确认通过：

```
go test ./internal/middleware/
```

预期输出：

```
ok  	go_ultra/internal/middleware	0.0xxs
```

- [ ] commit：

```
git add server/internal/middleware/middleware.go server/internal/middleware/middleware_test.go
git commit -m "feat(middleware): add RequestID, Logger and Recover middleware"
```

---

### Task 4: middleware —— PlayerAuth 与 AdminAuth

**Files:**
- Create: `server/internal/middleware/auth.go`
- Test: `server/internal/middleware/auth_test.go`

本 Task 实现两个鉴权中间件。它们需要访问 session 数据，但为避免直接依赖具体 service 类型造成耦合，定义两个最小接口（`PlayerSessionChecker` / `AdminSessionChecker`），由 `PlayerService` / `AdminService` 隐式实现。

- `PlayerAuth(checker)`：读 `go_ultra_session` cookie → `GetSession` → 注入 `playerID` 到 gin.Context；缺失/过期 → `domain.ErrNotAuthenticated`。
- `AdminAuth(checker)`：读 `go_ultra_admin` cookie → `CheckAdminSession`，失败 → `domain.ErrAdminRequired`。

> 契约里 `AdminService` 暴露 `CheckAdminSession(ctx, token) (bool, time.Time, error)`。玩家会话方面，session 的读取由 `PlayerService` 提供的方法承担；本中间件通过接口 `GetSession(ctx, token) (playerID int64, ok bool, err error)` 调用。该方法对应阶段 3 中 `PlayerService` 基于 sqlc `GetSession` 查询的封装（含过期判断与滑动续期）。若阶段 3 的方法名不同，在装配时用一个薄适配器满足本接口即可（见 Task 11 router 装配说明）。

为了让本 Task 自包含且可独立测试，中间件依赖接口而非具体类型；测试用 fake 实现。

步骤：

- [ ] 写失败测试 `server/internal/middleware/auth_test.go`，内容完整如下：

```go
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakePlayerChecker struct {
	playerID int64
	ok       bool
	err      error
	gotToken string
}

func (f *fakePlayerChecker) GetSession(ctx context.Context, token string) (int64, bool, error) {
	f.gotToken = token
	return f.playerID, f.ok, f.err
}

type fakeAdminChecker struct {
	ok        bool
	expiresAt time.Time
	err       error
	gotToken  string
}

func (f *fakeAdminChecker) CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error) {
	f.gotToken = token
	return f.ok, f.expiresAt, f.err
}

func decodeErrCode(t *testing.T, body []byte) string {
	t.Helper()
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &b); err != nil {
		t.Fatalf("invalid json %q: %v", string(body), err)
	}
	return b.Error.Code
}

func TestPlayerAuth_NoCookie_401(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(&fakePlayerChecker{}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if code := decodeErrCode(t, w.Body.Bytes()); code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %q, want NOT_AUTHENTICATED", code)
	}
}

func TestPlayerAuth_InvalidSession_401(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(&fakePlayerChecker{ok: false}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_session", Value: "expired-or-unknown"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if code := decodeErrCode(t, w.Body.Bytes()); code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %q, want NOT_AUTHENTICATED", code)
	}
}

func TestPlayerAuth_Valid_InjectsPlayerID(t *testing.T) {
	checker := &fakePlayerChecker{playerID: 42, ok: true}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(checker))
	var got int64
	r.GET("/", func(c *gin.Context) {
		if v, ok := c.Get(CtxPlayerID); ok {
			got, _ = v.(int64)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_session", Value: "good-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got != 42 {
		t.Fatalf("playerID = %d, want 42", got)
	}
	if checker.gotToken != "good-token" {
		t.Fatalf("checker received token %q, want good-token", checker.gotToken)
	}
}

func TestPlayerAuth_CheckerError_500(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(&fakePlayerChecker{err: context.DeadlineExceeded}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_session", Value: "x"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestAdminAuth_NoCookie_403(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(AdminAuth(&fakeAdminChecker{}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if code := decodeErrCode(t, w.Body.Bytes()); code != "ADMIN_REQUIRED" {
		t.Fatalf("code = %q, want ADMIN_REQUIRED", code)
	}
}

func TestAdminAuth_InvalidSession_403(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(AdminAuth(&fakeAdminChecker{ok: false}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_admin", Value: "nope"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestAdminAuth_Valid_Passes(t *testing.T) {
	checker := &fakeAdminChecker{ok: true, expiresAt: time.Now().Add(time.Minute)}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(AdminAuth(checker))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_admin", Value: "admin-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if checker.gotToken != "admin-token" {
		t.Fatalf("checker received token %q, want admin-token", checker.gotToken)
	}
}

// 确保 domain 错误码字符串与中间件使用一致（防御 typo）。
func TestAuth_DomainCodes(t *testing.T) {
	if domain.ErrNotAuthenticated.Code != "NOT_AUTHENTICATED" {
		t.Fatalf("unexpected NOT_AUTHENTICATED code: %q", domain.ErrNotAuthenticated.Code)
	}
	if domain.ErrAdminRequired.Code != "ADMIN_REQUIRED" {
		t.Fatalf("unexpected ADMIN_REQUIRED code: %q", domain.ErrAdminRequired.Code)
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/middleware/
```

预期失败（编译错误，`PlayerAuth`/`AdminAuth`/接口未定义）：

```
# go_ultra/internal/middleware [go_ultra/internal/middleware.test]
./auth_test.go:...: undefined: PlayerAuth
./auth_test.go:...: undefined: AdminAuth
FAIL	go_ultra/internal/middleware [build failed]
```

- [ ] 写最小实现 `server/internal/middleware/auth.go`，内容完整如下：

```go
package middleware

import (
	"context"
	"net/http"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/session"

	"github.com/gin-gonic/gin"
)

// PlayerSessionChecker 校验玩家会话 token，返回对应 playerID。
// ok=false 表示 token 无效或已过期（非系统错误）。
type PlayerSessionChecker interface {
	GetSession(ctx context.Context, token string) (playerID int64, ok bool, err error)
}

// AdminSessionChecker 校验管理员会话 token。
type AdminSessionChecker interface {
	CheckAdminSession(ctx context.Context, token string) (ok bool, expiresAt time.Time, err error)
}

// abort 把 domain.Error 写成统一 JSON 并终止链；非 domain.Error 一律 500。
func abort(c *gin.Context, err error) {
	if de, ok := err.(*domain.Error); ok && de != nil {
		c.AbortWithStatusJSON(de.Status, gin.H{
			"error": gin.H{"code": de.Code, "message": de.Message},
		})
		return
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{"code": domain.ErrInternal.Code, "message": domain.ErrInternal.Message},
	})
}

// PlayerAuth 要求请求带有效的玩家会话 cookie，并把 playerID 注入 context。
func PlayerAuth(checker PlayerSessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(session.PlayerCookieName)
		if err != nil || token == "" {
			abort(c, domain.ErrNotAuthenticated)
			return
		}
		playerID, ok, err := checker.GetSession(c.Request.Context(), token)
		if err != nil {
			abort(c, domain.ErrInternal.WithCause(err))
			return
		}
		if !ok {
			abort(c, domain.ErrNotAuthenticated)
			return
		}
		c.Set(CtxPlayerID, playerID)
		c.Next()
	}
}

// AdminAuth 要求请求带有效的管理员会话 cookie。
func AdminAuth(checker AdminSessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(session.AdminCookieName)
		if err != nil || token == "" {
			abort(c, domain.ErrAdminRequired)
			return
		}
		ok, _, err := checker.CheckAdminSession(c.Request.Context(), token)
		if err != nil {
			abort(c, domain.ErrInternal.WithCause(err))
			return
		}
		if !ok {
			abort(c, domain.ErrAdminRequired)
			return
		}
		c.Next()
	}
}
```

- [ ] 运行确认通过：

```
go test ./internal/middleware/
```

预期输出：

```
ok  	go_ultra/internal/middleware	0.0xxs
```

- [ ] commit：

```
git add server/internal/middleware/auth.go server/internal/middleware/auth_test.go
git commit -m "feat(middleware): add PlayerAuth and AdminAuth middleware"
```

---

### Task 5: handler 测试基础设施 —— testharness

**Files:**
- Create: `server/internal/handler/harness_test.go`

本 Task 不含生产代码，只搭建后续所有 handler HTTP 测试共享的测试基础设施：用临时 sqlite 文件 + 真实 service 装配出一个 `*gin.Engine`，并提供登录辅助函数。后续 Task 6–10 的测试都复用它。

> 该文件依赖 Task 11 才创建的 `NewRouter` / `Deps`。为遵循 TDD（先红后绿），本 Task 写出 harness 后运行测试，会因 `NewRouter` 未定义而编译失败 —— 这是预期的红灯；待 Task 11 实现 `NewRouter` 后，本文件连同 Task 6–10 一起转绿。因此本 Task 的 commit 与 Task 6–11 是一组渐进式 TDD（每个 Task 各自 commit 自己的测试，最后 Task 11 让整包变绿）。

> harness 用 `db.New` 打开临时文件库（契约规定 `db.New(path)` 会跑 goose 迁移）。`:memory:` 在多连接 `*sql.DB` 下每条连接是独立库，故这里用 `t.TempDir()` 下的文件库，确保所有连接共享同一份数据。

步骤：

- [ ] 写 `server/internal/handler/harness_test.go`，内容完整如下：

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go_ultra/internal/db"
	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/service"

	"github.com/rs/zerolog"
)

// testServer 持有一个真实装配的 router 与底层依赖，供 HTTP 测试使用。
type testServer struct {
	t      *testing.T
	router http.Handler
	deps   Deps
}

// newTestServer 用临时 sqlite 文件库 + 真实 service 构造一个 router。
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	q := sqlc.New(sqlDB)
	deps := Deps{
		Player:      service.NewPlayerService(q, sqlDB),
		Match:       service.NewMatchService(q, sqlDB),
		Leaderboard: service.NewLeaderboardService(q, sqlDB),
		Admin:       service.NewAdminService(q, sqlDB),
		Logger:      zerolog.Nop(),
	}
	return &testServer{
		t:      t,
		router: NewRouter(deps),
		deps:   deps,
	}
}

// do 发起一个请求并返回 recorder。body 为已序列化好的 JSON 字符串（可空）。
func (ts *testServer) do(method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	ts.t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	for _, ck := range cookies {
		r.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, r)
	return w
}

// login 以给定用户名登录（隐式注册），返回玩家会话 cookie。
func (ts *testServer) login(username string) *http.Cookie {
	ts.t.Helper()
	w := ts.do(http.MethodPost, "/api/login", `{"username":"`+username+`"}`)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		ts.t.Fatalf("login(%q) status = %d, body=%s", username, w.Code, w.Body.String())
	}
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "go_ultra_session" {
			return ck
		}
	}
	ts.t.Fatalf("login(%q) did not set go_ultra_session cookie", username)
	return nil
}

// adminLogin 取出首启生成的管理员明文密码并登录，返回 admin cookie。
func (ts *testServer) adminLogin(plaintext string) *http.Cookie {
	ts.t.Helper()
	w := ts.do(http.MethodPost, "/api/admin/login", `{"password":"`+plaintext+`"}`)
	if w.Code != http.StatusOK {
		ts.t.Fatalf("admin login status = %d, body=%s", w.Code, w.Body.String())
	}
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "go_ultra_admin" {
			return ck
		}
	}
	ts.t.Fatalf("admin login did not set go_ultra_admin cookie")
	return nil
}
```

- [ ] 运行确认失败（预期：`NewRouter`/`Deps` 未定义，红灯，符合 TDD）：

```
go test ./internal/handler/
```

预期失败输出：

```
# go_ultra/internal/handler [go_ultra/internal/handler.test]
./harness_test.go:...: undefined: Deps
./harness_test.go:...: undefined: NewRouter
FAIL	go_ultra/internal/handler [build failed]
```

- [ ] commit（提交测试脚手架；该红灯将在 Task 11 转绿）：

```
git add server/internal/handler/harness_test.go
git commit -m "test(handler): add http test harness with real services on temp sqlite"
```

---

### Task 6: handler —— health.go（GET /api/healthz）

**Files:**
- Create: `server/internal/handler/health.go`
- Test: `server/internal/handler/health_test.go`

本 Task 实现健康检查端点 `GET /api/healthz` → 200 `{"status":"ok"}`。它无鉴权、无依赖，是最简单的 handler，先做以打通 router 路径。

步骤：

- [ ] 写失败测试 `server/internal/handler/health_test.go`，内容完整如下：

```go
package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestHealthz(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/healthz", "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want ok", body["status"])
	}
}
```

- [ ] 运行确认失败（此时 `health` handler 与 `NewRouter` 均未定义，红灯）：

```
go test ./internal/handler/ -run TestHealthz
```

预期失败：编译错误（`NewRouter` 未定义，因 Task 11 尚未完成）或 `handleHealthz` 未定义。

- [ ] 写最小实现 `server/internal/handler/health.go`，内容完整如下：

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleHealthz 返回服务存活探针响应。
func handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
```

- [ ] 由于 `NewRouter` 尚未实现，本 Task 测试仍会因编译失败而无法转绿。这是预期的（health handler 已就绪，等待 Task 11 挂载）。运行确认实现本身可编译：

```
go build ./internal/handler/
```

预期输出：成功（无报错），说明 `health.go` 语法正确。

- [ ] commit：

```
git add server/internal/handler/health.go server/internal/handler/health_test.go
git commit -m "feat(handler): add healthz endpoint and test"
```

---

### Task 7: handler —— auth.go（login/logout/me + admin login/logout/status）

**Files:**
- Create: `server/internal/handler/auth.go`
- Test: `server/internal/handler/auth_test.go`

本 Task 实现鉴权相关 handler：
- `POST /api/login`：隐式注册 + 设置玩家 cookie，返回 `{player}`（已存在 200 / 新建 201）
- `POST /api/logout`：清除 cookie，204
- `GET /api/me`：返回当前登录玩家（需 PlayerAuth）
- `POST /api/admin/login`：校验密码 + 设置 admin cookie，返回 `{expires_at}`
- `POST /api/admin/logout`：清除 admin cookie，204
- `GET /api/admin/status`：返回 `{authed, expires_at?}`

> handler 通过 `Deps` 暴露的 service 调用。本 Task 还约定 handler 需要的"创建会话"能力：玩家登录后需写入 session。契约的 `PlayerService` 提供 `LoginOrCreate`；写 session 的具体方法在阶段 3 由 `PlayerService` 提供（基于 sqlc `CreateSession`）。本计划约定 handler 通过 `Deps.Player` 调用 `CreatePlayerSession(ctx, playerID) (token string, expiresAt time.Time, err error)`、`DeletePlayerSession(ctx, token) error`、`GetSession(ctx, token) (playerID int64, ok bool, err error)`。这些是 `PlayerService` 在阶段 3 已实现的会话方法（名称与签名以本契约 http 层需求为准；若阶段 3 命名不同，在 Task 11 用适配器对齐，handler 代码不变）。

为使本 Task 自包含，下面给出 handler 完整代码，并把它依赖的 service 方法集合显式声明在 `auth.go` 顶部注释中。

步骤：

- [ ] 写失败测试 `server/internal/handler/auth_test.go`，内容完整如下：

```go
package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestLogin_ImplicitRegister_SetsCookie(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodPost, "/api/login", `{"username":"alice"}`)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 200/201; body=%s", w.Code, w.Body.String())
	}
	var hasCookie bool
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "go_ultra_session" && ck.Value != "" {
			hasCookie = true
			if !ck.HttpOnly {
				t.Fatalf("session cookie not HttpOnly")
			}
		}
	}
	if !hasCookie {
		t.Fatalf("login did not set go_ultra_session cookie")
	}

	var body struct {
		Player struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
			Rating   int    `json:"rating"`
		} `json:"player"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Player.Username != "alice" {
		t.Fatalf("username = %q, want alice", body.Player.Username)
	}
	if body.Player.Rating != 1500 {
		t.Fatalf("rating = %d, want 1500", body.Player.Rating)
	}
}

func TestLogin_InvalidBody_400(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodPost, "/api/login", `{"username":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestMe_RequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/me", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %q, want NOT_AUTHENTICATED", body.Error.Code)
	}
}

func TestMe_WithCookie_ReturnsPlayer(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("bob")
	w := ts.do(http.MethodGet, "/api/me", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Player struct {
			Username string `json:"username"`
		} `json:"player"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Player.Username != "bob" {
		t.Fatalf("username = %q, want bob", body.Player.Username)
	}
}

func TestLogout_204(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("carol")
	w := ts.do(http.MethodPost, "/api/logout", "", cookie)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminStatus_Unauthed(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/admin/status", "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Authed bool `json:"authed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Authed {
		t.Fatalf("authed = true, want false")
	}
}

func TestAdminLogin_WrongPassword_400(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodPost, "/api/admin/login", `{"password":"definitely-wrong"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/handler/ -run TestLogin
```

预期失败：编译错误（handler 函数与 `NewRouter` 未定义）。

- [ ] 写最小实现 `server/internal/handler/auth.go`，内容完整如下：

```go
package handler

import (
	"context"
	"net/http"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/session"

	"github.com/gin-gonic/gin"
)

// authPlayerService 是 auth handler 依赖的玩家会话相关能力集合。
// 由 *service.PlayerService 实现。
type authPlayerService interface {
	LoginOrCreate(ctx context.Context, username string) (domain.Player, error)
	GetByUsername(ctx context.Context, username string) (domain.Player, error)
	CreatePlayerSession(ctx context.Context, playerID int64) (token string, expiresAt time.Time, err error)
	DeletePlayerSession(ctx context.Context, token string) error
}

// authAdminService 是 admin 鉴权 handler 依赖的能力集合。由 *service.AdminService 实现。
type authAdminService interface {
	VerifyPassword(ctx context.Context, pw string) (bool, error)
	CreateAdminSession(ctx context.Context) (token string, expiresAt time.Time, err error)
	CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error)
	DeleteAdminSession(ctx context.Context, token string) error
}

type loginRequest struct {
	Username string `json:"username"`
}

type adminLoginRequest struct {
	Password string `json:"password"`
}

// playerDTO 是 player 的 JSON 表示（snake_case）。
type playerDTO struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Rating    int    `json:"rating"`
	Dan       int    `json:"dan"`
	CreatedAt string `json:"created_at"`
}

func toPlayerDTO(p domain.Player) playerDTO {
	return playerDTO{
		ID:        p.ID,
		Username:  p.Username,
		Rating:    p.Rating,
		Dan:       domain.Dan(p.Rating),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// setPlayerCookie 写入玩家会话 cookie。
func setPlayerCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.PlayerCookieName, token, int(ttl.Seconds()), "/", "", true, true)
}

func clearPlayerCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.PlayerCookieName, "", -1, "/", "", true, true)
}

func setAdminCookie(c *gin.Context, token string, ttl time.Duration) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.AdminCookieName, token, int(ttl.Seconds()), "/", "", true, true)
}

func clearAdminCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(session.AdminCookieName, "", -1, "/", "", true, true)
}

type authHandler struct {
	player authPlayerService
	admin  authAdminService
}

// handleLogin 隐式注册或登录玩家，并设置会话 cookie。
func (h *authHandler) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}

	// 判断是否为新建：登录前先查是否已存在。
	existed := true
	if _, err := h.player.GetByUsername(c.Request.Context(), req.Username); err != nil {
		if de, ok := err.(*domain.Error); ok && de.Code == domain.ErrPlayerNotFound.Code {
			existed = false
		}
		// 其它错误（含校验类）交由 LoginOrCreate 统一处理。
	}

	p, err := h.player.LoginOrCreate(c.Request.Context(), req.Username)
	if err != nil {
		respondError(c, err)
		return
	}

	token, expiresAt, err := h.player.CreatePlayerSession(c.Request.Context(), p.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	setPlayerCookie(c, token, time.Until(expiresAt))

	status := http.StatusOK
	if !existed {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"player": toPlayerDTO(p)})
}

// handleLogout 删除当前会话并清除 cookie。
func (h *authHandler) handleLogout(c *gin.Context) {
	if token, err := c.Cookie(session.PlayerCookieName); err == nil && token != "" {
		_ = h.player.DeletePlayerSession(c.Request.Context(), token)
	}
	clearPlayerCookie(c)
	c.Status(http.StatusNoContent)
}

// handleMe 返回当前登录玩家信息（依赖 PlayerAuth 注入的 playerID）。
func (h *authHandler) handleMe(c *gin.Context) {
	username, ok := currentUsername(c, h.player)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"player": username})
}

// currentUsername 取出当前玩家并返回其 DTO；失败时已写好响应，返回 ok=false。
func currentUsername(c *gin.Context, p authPlayerService) (playerDTO, bool) {
	v, exists := c.Get(ctxPlayerID)
	if !exists {
		respondError(c, domain.ErrNotAuthenticated)
		return playerDTO{}, false
	}
	pid, _ := v.(int64)
	pl, err := p.GetByID(c.Request.Context(), pid)
	if err != nil {
		respondError(c, err)
		return playerDTO{}, false
	}
	return toPlayerDTO(pl), true
}

// handleAdminLogin 校验密码并创建管理员会话。
func (h *authHandler) handleAdminLogin(c *gin.Context) {
	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}
	ok, err := h.admin.VerifyPassword(c.Request.Context(), req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		respondError(c, domain.ErrInvalidParam)
		return
	}
	token, expiresAt, err := h.admin.CreateAdminSession(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	setAdminCookie(c, token, time.Until(expiresAt))
	c.JSON(http.StatusOK, gin.H{"expires_at": expiresAt.UTC().Format(time.RFC3339)})
}

// handleAdminLogout 删除管理员会话并清除 cookie。
func (h *authHandler) handleAdminLogout(c *gin.Context) {
	if token, err := c.Cookie(session.AdminCookieName); err == nil && token != "" {
		_ = h.admin.DeleteAdminSession(c.Request.Context(), token)
	}
	clearAdminCookie(c)
	c.Status(http.StatusNoContent)
}

// handleAdminStatus 返回当前管理员会话状态（无需 AdminAuth）。
func (h *authHandler) handleAdminStatus(c *gin.Context) {
	token, err := c.Cookie(session.AdminCookieName)
	if err != nil || token == "" {
		c.JSON(http.StatusOK, gin.H{"authed": false})
		return
	}
	ok, expiresAt, err := h.admin.CheckAdminSession(c.Request.Context(), token)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		c.JSON(http.StatusOK, gin.H{"authed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"authed":     true,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}
```

> 注意：`handleMe` 用到的 `GetByID` 与 `currentUsername` 调用的 `p.GetByID` 需要 `authPlayerService` 接口包含该方法。下面在接口里补充（见下一步的编译修正）。

- [ ] 修正接口：把 `authPlayerService` 接口加上 `GetByID`，因为 `currentUsername` 需要它。用 Edit 把接口定义改为包含 `GetByID`。将 `auth.go` 中

```go
type authPlayerService interface {
	LoginOrCreate(ctx context.Context, username string) (domain.Player, error)
	GetByUsername(ctx context.Context, username string) (domain.Player, error)
	CreatePlayerSession(ctx context.Context, playerID int64) (token string, expiresAt time.Time, err error)
	DeletePlayerSession(ctx context.Context, token string) error
}
```

替换为

```go
type authPlayerService interface {
	LoginOrCreate(ctx context.Context, username string) (domain.Player, error)
	GetByUsername(ctx context.Context, username string) (domain.Player, error)
	GetByID(ctx context.Context, playerID int64) (domain.Player, error)
	CreatePlayerSession(ctx context.Context, playerID int64) (token string, expiresAt time.Time, err error)
	DeletePlayerSession(ctx context.Context, token string) error
}
```

> 说明：`GetByID` 是 `PlayerService` 在阶段 3 已存在的便利方法（基于 sqlc `GetPlayerByID`）。`GetStats`、`ListByRating` 等在后续 Task 的接口中按需声明。

- [ ] 运行确认实现可编译（router 尚未实现，整包测试仍编译失败，但 build 单包验证 auth.go 语法）：

```
go build ./internal/handler/
```

预期：成功（无报错）。若有报错按报错修正后再继续。

- [ ] commit：

```
git add server/internal/handler/auth.go server/internal/handler/auth_test.go
git commit -m "feat(handler): add auth handlers (login/logout/me + admin login/logout/status)"
```

---

### Task 8: handler —— player.go（玩家查询端点）

**Files:**
- Create: `server/internal/handler/player.go`
- Test: `server/internal/handler/player_test.go`

本 Task 实现玩家相关查询端点（均需 PlayerAuth）：
- `GET /api/players` → 玩家列表（含 dan、games_played、win_rate）
- `GET /api/players/:username` → 单玩家 + stats
- `GET /api/players/:username/history?from=&to=` → 历史曲线点
- `GET /api/players/:username/matches?limit=&offset=` → 该玩家对局流

步骤：

- [ ] 写失败测试 `server/internal/handler/player_test.go`，内容完整如下：

```go
package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListPlayers_RequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/players", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestListPlayers_ReturnsArray(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")

	w := ts.do(http.MethodGet, "/api/players", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var arr []struct {
		Username string `json:"username"`
		Rating   int    `json:"rating"`
		Dan      int    `json:"dan"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(arr) != 2 {
		t.Fatalf("len = %d, want 2; body=%s", len(arr), w.Body.String())
	}
}

func TestGetPlayer_ByUsername(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")

	w := ts.do(http.MethodGet, "/api/players/alice", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Username string `json:"username"`
		Stats    struct {
			Wins   int `json:"wins"`
			Losses int `json:"losses"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if body.Username != "alice" {
		t.Fatalf("username = %q, want alice", body.Username)
	}
}

func TestGetPlayer_NotFound_404(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/players/ghost", "", cookie)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "PLAYER_NOT_FOUND" {
		t.Fatalf("code = %q, want PLAYER_NOT_FOUND", body.Error.Code)
	}
}

func TestPlayerHistory_HasStartingPoint(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/players/alice/history", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var pts []struct {
		PlayedAt string `json:"played_at"`
		Rating   int    `json:"rating"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &pts); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	// 没有对局时也应至少含 1 个起点（created_at, 1500）。
	if len(pts) < 1 {
		t.Fatalf("history len = %d, want >= 1", len(pts))
	}
	if pts[0].Rating != 1500 {
		t.Fatalf("first point rating = %d, want 1500", pts[0].Rating)
	}
}

func TestPlayerMatches_Empty(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/players/alice/matches", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var arr []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(arr) != 0 {
		t.Fatalf("matches len = %d, want 0", len(arr))
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/handler/ -run TestListPlayers
```

预期失败：编译错误（player handler 与 `NewRouter` 未定义）。

- [ ] 写最小实现 `server/internal/handler/player.go`，内容完整如下：

```go
package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/service"

	"github.com/gin-gonic/gin"
)

// playerSvc 是 player handler 依赖的能力集合。由 *service.PlayerService 实现。
type playerSvc interface {
	GetByUsername(ctx context.Context, username string) (domain.Player, error)
	GetStats(ctx context.Context, playerID int64) (domain.Stats, error)
	ListByRating(ctx context.Context) ([]domain.Player, error)
}

// playerMatchSvc 是 player handler 依赖的对局相关能力。由 *service.MatchService 实现。
type playerMatchSvc interface {
	ListByPlayer(ctx context.Context, playerID int64, limit, offset int) ([]service.MatchView, error)
	History(ctx context.Context, playerID int64, createdAt time.Time) ([]service.HistoryPoint, error)
}

type playerHandler struct {
	player playerSvc
	match  playerMatchSvc
	// statsCounter 用于列表页拿每个玩家的胜负数，复用 player service 的 GetStats。
}

type playerListItem struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	Rating      int     `json:"rating"`
	Dan         int     `json:"dan"`
	GamesPlayed int     `json:"games_played"`
	WinRate     float64 `json:"win_rate"`
}

type playerDetail struct {
	ID        int64        `json:"id"`
	Username  string       `json:"username"`
	Rating    int          `json:"rating"`
	Dan       int          `json:"dan"`
	CreatedAt string       `json:"created_at"`
	Stats     playerStats  `json:"stats"`
}

type playerStats struct {
	Wins          int     `json:"wins"`
	Losses        int     `json:"losses"`
	WinRate       float64 `json:"win_rate"`
	CurrentStreak int     `json:"current_streak"`
	LongestStreak int     `json:"longest_streak"`
}

type historyPointDTO struct {
	PlayedAt string `json:"played_at"`
	Rating   int    `json:"rating"`
}

type matchViewDTO struct {
	ID           int64  `json:"id"`
	Opponent     string `json:"opponent"`
	Result       string `json:"result"`
	RatingBefore int    `json:"rating_before"`
	RatingAfter  int    `json:"rating_after"`
	Delta        int    `json:"delta"`
	PlayedAt     string `json:"played_at"`
}

// handleListPlayers 返回所有玩家（按 rating DESC）及其统计。
func (h *playerHandler) handleListPlayers(c *gin.Context) {
	players, err := h.player.ListByRating(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]playerListItem, 0, len(players))
	for _, p := range players {
		st, err := h.player.GetStats(c.Request.Context(), p.ID)
		if err != nil {
			respondError(c, err)
			return
		}
		out = append(out, playerListItem{
			ID:          p.ID,
			Username:    p.Username,
			Rating:      p.Rating,
			Dan:         domain.Dan(p.Rating),
			GamesPlayed: st.Wins + st.Losses,
			WinRate:     st.WinRate,
		})
	}
	c.JSON(http.StatusOK, out)
}

// handleGetPlayer 返回单个玩家及其完整统计。
func (h *playerHandler) handleGetPlayer(c *gin.Context) {
	username := c.Param("username")
	p, err := h.player.GetByUsername(c.Request.Context(), username)
	if err != nil {
		respondError(c, err)
		return
	}
	st, err := h.player.GetStats(c.Request.Context(), p.ID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, playerDetail{
		ID:        p.ID,
		Username:  p.Username,
		Rating:    p.Rating,
		Dan:       domain.Dan(p.Rating),
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339),
		Stats: playerStats{
			Wins:          st.Wins,
			Losses:        st.Losses,
			WinRate:       st.WinRate,
			CurrentStreak: st.CurrentStreak,
			LongestStreak: st.LongestStreak,
		},
	})
}

// handlePlayerHistory 返回玩家历史曲线点（service 已 prepend 起点）。
func (h *playerHandler) handlePlayerHistory(c *gin.Context) {
	username := c.Param("username")
	p, err := h.player.GetByUsername(c.Request.Context(), username)
	if err != nil {
		respondError(c, err)
		return
	}
	pts, err := h.match.History(c.Request.Context(), p.ID, p.CreatedAt)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]historyPointDTO, 0, len(pts))
	for _, pt := range pts {
		out = append(out, historyPointDTO{
			PlayedAt: pt.PlayedAt.UTC().Format(time.RFC3339),
			Rating:   pt.Rating,
		})
	}
	c.JSON(http.StatusOK, out)
}

// handlePlayerMatches 返回玩家对局流（分页）。
func (h *playerHandler) handlePlayerMatches(c *gin.Context) {
	username := c.Param("username")
	p, err := h.player.GetByUsername(c.Request.Context(), username)
	if err != nil {
		respondError(c, err)
		return
	}
	limit, offset := parseLimitOffset(c)
	views, err := h.match.ListByPlayer(c.Request.Context(), p.ID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]matchViewDTO, 0, len(views))
	for _, v := range views {
		out = append(out, toMatchViewDTO(v))
	}
	c.JSON(http.StatusOK, out)
}

func toMatchViewDTO(v service.MatchView) matchViewDTO {
	return matchViewDTO{
		ID:           v.ID,
		Opponent:     v.Opponent,
		Result:       v.Result,
		RatingBefore: v.RatingBefore,
		RatingAfter:  v.RatingAfter,
		Delta:        v.Delta,
		PlayedAt:     v.PlayedAt.UTC().Format(time.RFC3339),
	}
}

// parseLimitOffset 解析 limit/offset 查询参数，提供默认值与上限。
func parseLimitOffset(c *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
			if limit > 500 {
				limit = 500
			}
		}
	}
	if s := c.Query("offset"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
```

- [ ] 运行确认实现可编译：

```
go build ./internal/handler/
```

预期：成功。

- [ ] commit：

```
git add server/internal/handler/player.go server/internal/handler/player_test.go
git commit -m "feat(handler): add player query endpoints (list/detail/history/matches)"
```

---

### Task 9: handler —— match.go（录入、列表、软删除、恢复）

**Files:**
- Create: `server/internal/handler/match.go`
- Test: `server/internal/handler/match_test.go`

本 Task 实现对局端点：
- `POST /api/matches`：`played_at` 可选默认 now；拒绝未来时间 → `ErrInvalidParam`；self → `ErrSelfMatch`（由 service 抛出）；返回 201 + RecordResult
- `GET /api/matches?limit=&offset=`：全局对局流
- `DELETE /api/matches/:id`：走 AdminAuth，软删除，204
- `GET /api/admin/matches/deleted`：已删除列表（AdminAuth）
- `POST /api/admin/matches/:id/restore`：恢复，204（AdminAuth）

步骤：

- [ ] 写失败测试 `server/internal/handler/match_test.go`，内容完整如下：

```go
package handler

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestRecordMatch_Self_409(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	body := `{"opponent_username":"alice","result":"win"}`
	w := ts.do(http.MethodPost, "/api/matches", body, cookie)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "SELF_MATCH" {
		t.Fatalf("code = %q, want SELF_MATCH", b.Error.Code)
	}
}

func TestRecordMatch_FuturePlayedAt_400(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	future := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	body := `{"opponent_username":"bob","result":"win","played_at":"` + future + `"}`
	w := ts.do(http.MethodPost, "/api/matches", body, cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "INVALID_PARAM" {
		t.Fatalf("code = %q, want INVALID_PARAM", b.Error.Code)
	}
}

func TestRecordMatch_Success_201(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	body := `{"opponent_username":"bob","result":"win"}`
	w := ts.do(http.MethodPost, "/api/matches", body, cookie)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var res struct {
		ID                int64 `json:"id"`
		WinnerDelta       int   `json:"winner_delta"`
		LoserDelta        int   `json:"loser_delta"`
		NewSelfRating     int   `json:"new_self_rating"`
		NewOpponentRating int   `json:"new_opponent_rating"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if res.WinnerDelta+res.LoserDelta != 0 {
		t.Fatalf("delta not zero-sum: %d + %d", res.WinnerDelta, res.LoserDelta)
	}
	if res.NewSelfRating <= 1500 {
		t.Fatalf("winner new rating = %d, want > 1500", res.NewSelfRating)
	}
}

func TestListGlobalMatches(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	_ = ts.do(http.MethodPost, "/api/matches", `{"opponent_username":"bob","result":"win"}`, cookie)

	w := ts.do(http.MethodGet, "/api/matches", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var arr []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(arr) != 1 {
		t.Fatalf("global matches len = %d, want 1; body=%s", len(arr), w.Body.String())
	}
}

func TestDeleteMatch_RequiresAdmin_403(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodDelete, "/api/matches/1", "", cookie)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "ADMIN_REQUIRED" {
		t.Fatalf("code = %q, want ADMIN_REQUIRED", b.Error.Code)
	}
}

func TestDeleteMatch_NoCookie_403(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodDelete, "/api/matches/1", "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", w.Code, w.Body.String())
	}
}

func TestAdminDeleteAndRestoreFlow(t *testing.T) {
	ts := newTestServer(t)
	pw := ts.deps.adminPlaintext // 由 EnsurePassword 注入（见 harness 扩展）
	if pw == "" {
		t.Skip("admin plaintext not available in this harness build")
	}
	playerCookie := ts.login("alice")
	_ = ts.login("bob")
	rec := ts.do(http.MethodPost, "/api/matches", `{"opponent_username":"bob","result":"win"}`, playerCookie)
	var created struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	adminCookie := ts.adminLogin(pw)

	// 删除
	del := ts.do(http.MethodDelete, "/api/matches/1", "", adminCookie)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", del.Code, del.Body.String())
	}

	// 全局列表应为空
	list := ts.do(http.MethodGet, "/api/matches", "", playerCookie)
	var arr []map[string]any
	_ = json.Unmarshal(list.Body.Bytes(), &arr)
	if len(arr) != 0 {
		t.Fatalf("global matches after delete = %d, want 0", len(arr))
	}

	// 已删除列表应有 1 条
	deleted := ts.do(http.MethodGet, "/api/admin/matches/deleted", "", adminCookie)
	if deleted.Code != http.StatusOK {
		t.Fatalf("deleted list status = %d, want 200; body=%s", deleted.Code, deleted.Body.String())
	}

	// 恢复
	restore := ts.do(http.MethodPost, "/api/admin/matches/1/restore", "", adminCookie)
	if restore.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204; body=%s", restore.Code, restore.Body.String())
	}
}
```

> 上面 `ts.deps.adminPlaintext` 需要 harness 暴露首启明文密码。下一步先扩展 harness，再写 match handler。

- [ ] 扩展 harness 暴露管理员明文：用 Edit 在 `server/internal/handler/harness_test.go` 的 `testServer` 结构体与 `newTestServer` 中增加字段与赋值。

把

```go
type testServer struct {
	t      *testing.T
	router http.Handler
	deps   Deps
}
```

替换为

```go
type testServer struct {
	t              *testing.T
	router         http.Handler
	deps           Deps
	adminPlaintext string
}
```

并把

```go
	q := sqlc.New(sqlDB)
	deps := Deps{
		Player:      service.NewPlayerService(q, sqlDB),
		Match:       service.NewMatchService(q, sqlDB),
		Leaderboard: service.NewLeaderboardService(q, sqlDB),
		Admin:       service.NewAdminService(q, sqlDB),
		Logger:      zerolog.Nop(),
	}
	return &testServer{
		t:      t,
		router: NewRouter(deps),
		deps:   deps,
	}
}
```

替换为

```go
	q := sqlc.New(sqlDB)
	adminSvc := service.NewAdminService(q, sqlDB)
	plaintext, _, err := adminSvc.EnsurePassword(context.Background())
	if err != nil {
		t.Fatalf("EnsurePassword failed: %v", err)
	}
	deps := Deps{
		Player:      service.NewPlayerService(q, sqlDB),
		Match:       service.NewMatchService(q, sqlDB),
		Leaderboard: service.NewLeaderboardService(q, sqlDB),
		Admin:       adminSvc,
		Logger:      zerolog.Nop(),
	}
	return &testServer{
		t:              t,
		router:         NewRouter(deps),
		deps:           deps,
		adminPlaintext: plaintext,
	}
}
```

并在 `harness_test.go` 的 import 块加入 `"context"`（把 import 第一行 `"net/http"` 之前补一行）。将

```go
import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
```

替换为

```go
import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
```

- [ ] 由于把 `adminPlaintext` 提供给了测试，移除 match_test.go 中的 `t.Skip` 分支。将

```go
	pw := ts.deps.adminPlaintext // 由 EnsurePassword 注入（见 harness 扩展）
	if pw == "" {
		t.Skip("admin plaintext not available in this harness build")
	}
```

替换为

```go
	pw := ts.adminPlaintext
	if pw == "" {
		t.Fatalf("admin plaintext not generated")
	}
```

- [ ] 运行确认失败：

```
go test ./internal/handler/ -run TestRecordMatch
```

预期失败：编译错误（match handler 与 `NewRouter` 未定义）。

- [ ] 写最小实现 `server/internal/handler/match.go`，内容完整如下：

```go
package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/service"

	"github.com/gin-gonic/gin"
)

// matchSvc 是 match handler 依赖的录入/查询能力。由 *service.MatchService 实现。
type matchSvc interface {
	Record(ctx context.Context, submitterID int64, opponentUsername string, result string, playedAt time.Time) (service.RecordResult, error)
	ListGlobal(ctx context.Context, limit, offset int) ([]service.MatchView, error)
}

// adminMatchSvc 是删除/恢复/已删除列表能力。由 *service.AdminService 实现。
type adminMatchSvc interface {
	SoftDelete(ctx context.Context, matchID int64) error
	Restore(ctx context.Context, matchID int64) error
	ListDeleted(ctx context.Context) ([]domain.Match, error)
}

type matchHandler struct {
	match matchSvc
	admin adminMatchSvc
}

type recordMatchRequest struct {
	OpponentUsername string  `json:"opponent_username"`
	Result           string  `json:"result"`
	PlayedAt         *string `json:"played_at"`
}

type recordResultDTO struct {
	ID                int64 `json:"id"`
	WinnerDelta       int   `json:"winner_delta"`
	LoserDelta        int   `json:"loser_delta"`
	NewSelfRating     int   `json:"new_self_rating"`
	NewOpponentRating int   `json:"new_opponent_rating"`
}

type deletedMatchDTO struct {
	ID        int64  `json:"id"`
	WinnerID  int64  `json:"winner_id"`
	LoserID   int64  `json:"loser_id"`
	PlayedAt  string `json:"played_at"`
	DeletedAt string `json:"deleted_at"`
}

// handleRecordMatch 录入一局对局。
func (h *matchHandler) handleRecordMatch(c *gin.Context) {
	v, exists := c.Get(ctxPlayerID)
	if !exists {
		respondError(c, domain.ErrNotAuthenticated)
		return
	}
	submitterID, _ := v.(int64)

	var req recordMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}
	if req.Result != "win" && req.Result != "loss" {
		respondError(c, domain.ErrInvalidBody)
		return
	}

	playedAt := time.Now().UTC()
	if req.PlayedAt != nil && *req.PlayedAt != "" {
		t, err := time.Parse(time.RFC3339, *req.PlayedAt)
		if err != nil {
			respondError(c, domain.ErrInvalidParam.WithCause(err))
			return
		}
		playedAt = t.UTC()
		if playedAt.After(time.Now().UTC()) {
			respondError(c, domain.ErrInvalidParam)
			return
		}
	}

	res, err := h.match.Record(c.Request.Context(), submitterID, req.OpponentUsername, req.Result, playedAt)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, recordResultDTO{
		ID:                res.MatchID,
		WinnerDelta:       res.WinnerDelta,
		LoserDelta:        res.LoserDelta,
		NewSelfRating:     res.NewSelfRating,
		NewOpponentRating: res.NewOpponentRating,
	})
}

// handleListGlobalMatches 返回全局对局流（不含已删除）。
func (h *matchHandler) handleListGlobalMatches(c *gin.Context) {
	limit, offset := parseLimitOffset(c)
	views, err := h.match.ListGlobal(c.Request.Context(), limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]matchViewDTO, 0, len(views))
	for _, v := range views {
		out = append(out, toMatchViewDTO(v))
	}
	c.JSON(http.StatusOK, out)
}

// handleDeleteMatch 软删除一局（仅管理员）。
func (h *matchHandler) handleDeleteMatch(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.admin.SoftDelete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// handleListDeletedMatches 返回已删除对局列表（仅管理员）。
func (h *matchHandler) handleListDeletedMatches(c *gin.Context) {
	matches, err := h.admin.ListDeleted(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]deletedMatchDTO, 0, len(matches))
	for _, m := range matches {
		dto := deletedMatchDTO{
			ID:       m.ID,
			WinnerID: m.WinnerID,
			LoserID:  m.LoserID,
			PlayedAt: m.PlayedAt.UTC().Format(time.RFC3339),
		}
		if m.DeletedAt != nil {
			dto.DeletedAt = m.DeletedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, dto)
	}
	c.JSON(http.StatusOK, out)
}

// handleRestoreMatch 恢复一局软删除（仅管理员）。
func (h *matchHandler) handleRestoreMatch(c *gin.Context) {
	id, err := parseIDParam(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.admin.Restore(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// parseIDParam 解析路径参数 :id。
func parseIDParam(c *gin.Context) (int64, error) {
	s := c.Param("id")
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrInvalidParam
	}
	return id, nil
}
```

> 说明：`adminMatchSvc` 接口要求 `AdminService.ListDeleted(ctx) ([]domain.Match, error)`。契约 AdminService 列出的方法为 `SoftDelete`/`Restore` 等，已删除列表对应 sqlc `ListDeletedMatches`；按本契约 http 层需求，`AdminService` 在阶段 3 应提供 `ListDeleted`。若阶段 3 命名为别的，在 Task 11 用适配器对齐。

- [ ] 运行确认实现可编译：

```
go build ./internal/handler/
```

预期：成功。

- [ ] commit：

```
git add server/internal/handler/match.go server/internal/handler/match_test.go server/internal/handler/harness_test.go
git commit -m "feat(handler): add match endpoints (record/list/delete/restore) with future-played_at and self-match guards"
```

---

### Task 10: handler —— leaderboard.go（排行榜 + 对比）

**Files:**
- Create: `server/internal/handler/leaderboard.go`
- Test: `server/internal/handler/leaderboard_test.go`

本 Task 实现：
- `GET /api/leaderboard?min_games=` → 排行榜（默认 min_games=0）
- `GET /api/compare?usernames=a,b,c` → 多人对比；usernames 上限 10，超出 `ErrInvalidParam`

步骤：

- [ ] 写失败测试 `server/internal/handler/leaderboard_test.go`，内容完整如下：

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestLeaderboard_RequiresAuth(t *testing.T) {
	ts := newTestServer(t)
	w := ts.do(http.MethodGet, "/api/leaderboard", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", w.Code, w.Body.String())
	}
}

func TestLeaderboard_ReturnsRanks(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	_ = ts.do(http.MethodPost, "/api/matches", `{"opponent_username":"bob","result":"win"}`, cookie)

	w := ts.do(http.MethodGet, "/api/leaderboard?min_games=0", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var rows []struct {
		Rank     int    `json:"rank"`
		Username string `json:"username"`
		Rating   int    `json:"rating"`
		Dan      int    `json:"dan"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(rows) < 1 {
		t.Fatalf("rows len = %d, want >= 1", len(rows))
	}
	if rows[0].Rank != 1 {
		t.Fatalf("first row rank = %d, want 1", rows[0].Rank)
	}
}

func TestCompare_TooMany_400(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	names := make([]string, 11)
	for i := range names {
		names[i] = "u" + string(rune('a'+i))
	}
	w := ts.do(http.MethodGet, "/api/compare?usernames="+strings.Join(names, ","), "", cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if b.Error.Code != "INVALID_PARAM" {
		t.Fatalf("code = %q, want INVALID_PARAM", b.Error.Code)
	}
}

func TestCompare_Empty_400(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	w := ts.do(http.MethodGet, "/api/compare", "", cookie)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestCompare_Valid(t *testing.T) {
	ts := newTestServer(t)
	cookie := ts.login("alice")
	_ = ts.login("bob")
	w := ts.do(http.MethodGet, "/api/compare?usernames=alice,bob", "", cookie)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Series []struct {
			Username string `json:"username"`
		} `json:"series"`
		HeadToHead []map[string]any `json:"head_to_head"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json %q: %v", w.Body.String(), err)
	}
	if len(body.Series) != 2 {
		t.Fatalf("series len = %d, want 2", len(body.Series))
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/handler/ -run TestLeaderboard
```

预期失败：编译错误（leaderboard handler 与 `NewRouter` 未定义）。

- [ ] 写最小实现 `server/internal/handler/leaderboard.go`，内容完整如下：

```go
package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go_ultra/internal/domain"
	"go_ultra/internal/service"

	"github.com/gin-gonic/gin"
)

// leaderboardSvc 是 leaderboard handler 依赖的能力。由 *service.LeaderboardService 实现。
type leaderboardSvc interface {
	List(ctx context.Context, minGames int) ([]service.LeaderboardRow, error)
	CompareData(ctx context.Context, usernames []string) (service.CompareResult, error)
}

type leaderboardHandler struct {
	svc leaderboardSvc
}

type leaderboardRowDTO struct {
	Rank        int     `json:"rank"`
	Username    string  `json:"username"`
	Rating      int     `json:"rating"`
	Dan         int     `json:"dan"`
	GamesPlayed int     `json:"games_played"`
	WinRate     float64 `json:"win_rate"`
}

type comparePointDTO struct {
	PlayedAt string `json:"played_at"`
	Rating   int    `json:"rating"`
}

type compareSeriesDTO struct {
	Username string            `json:"username"`
	Color    string            `json:"color"`
	Points   []comparePointDTO `json:"points"`
}

type headToHeadDTO struct {
	A     string `json:"a"`
	B     string `json:"b"`
	AWins int    `json:"a_wins"`
	BWins int    `json:"b_wins"`
}

type compareResponse struct {
	Series     []compareSeriesDTO `json:"series"`
	HeadToHead []headToHeadDTO    `json:"head_to_head"`
}

// handleLeaderboard 返回排行榜。
func (h *leaderboardHandler) handleLeaderboard(c *gin.Context) {
	minGames := 0
	if s := c.Query("min_games"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			minGames = n
		}
	}
	rows, err := h.svc.List(c.Request.Context(), minGames)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]leaderboardRowDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, leaderboardRowDTO{
			Rank:        r.Rank,
			Username:    r.Username,
			Rating:      r.Rating,
			Dan:         r.Dan,
			GamesPlayed: r.GamesPlayed,
			WinRate:     r.WinRate,
		})
	}
	c.JSON(http.StatusOK, out)
}

// handleCompare 返回多人对比数据。usernames 上限 10。
func (h *leaderboardHandler) handleCompare(c *gin.Context) {
	raw := c.Query("usernames")
	names := splitUsernames(raw)
	if len(names) == 0 {
		respondError(c, domain.ErrInvalidParam)
		return
	}
	if len(names) > 10 {
		respondError(c, domain.ErrInvalidParam)
		return
	}
	res, err := h.svc.CompareData(c.Request.Context(), names)
	if err != nil {
		respondError(c, err)
		return
	}

	series := make([]compareSeriesDTO, 0, len(res.Series))
	for _, s := range res.Series {
		pts := make([]comparePointDTO, 0, len(s.Points))
		for _, p := range s.Points {
			pts = append(pts, comparePointDTO{
				PlayedAt: p.PlayedAt.UTC().Format(time.RFC3339),
				Rating:   p.Rating,
			})
		}
		series = append(series, compareSeriesDTO{
			Username: s.Username,
			Color:    s.Color,
			Points:   pts,
		})
	}
	h2h := make([]headToHeadDTO, 0, len(res.HeadToHead))
	for _, x := range res.HeadToHead {
		h2h = append(h2h, headToHeadDTO{A: x.A, B: x.B, AWins: x.AWins, BWins: x.BWins})
	}
	c.JSON(http.StatusOK, compareResponse{Series: series, HeadToHead: h2h})
}

// splitUsernames 把逗号分隔的字符串拆成去重后的非空用户名列表。
func splitUsernames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
```

- [ ] 运行确认实现可编译：

```
go build ./internal/handler/
```

预期：成功。

- [ ] commit：

```
git add server/internal/handler/leaderboard.go server/internal/handler/leaderboard_test.go
git commit -m "feat(handler): add leaderboard and compare endpoints with usernames<=10 guard"
```

---

### Task 11: router.go —— NewRouter(Deps) 装配全部中间件与路由

**Files:**
- Create: `server/internal/handler/router.go`
- Test:（无新增测试文件；本 Task 让 Task 5–10 的整包测试全部转绿）

本 Task 实现 `NewRouter(deps Deps) *gin.Engine`，挂载全局中间件（RequestID → Logger → Recover）与全部路由，按 spec §6.1 严格映射，并对需要鉴权的分组应用 `PlayerAuth` / `AdminAuth`。这是把前面所有 handler 串起来、让红灯转绿的关键一步。

> 鉴权中间件需要 `PlayerSessionChecker` / `AdminSessionChecker`。`Deps.Player`（`*service.PlayerService`）实现 `GetSession`，`Deps.Admin`（`*service.AdminService`）实现 `CheckAdminSession`，二者直接满足 middleware 的接口。各 handler 结构体的接口方法集合也都由对应 service 满足。

步骤：

- [ ] 写实现 `server/internal/handler/router.go`，内容完整如下：

```go
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"go_ultra/internal/middleware"
	"go_ultra/internal/service"
)

// Deps 是装配 router 所需的全部依赖。
type Deps struct {
	Player      *service.PlayerService
	Match       *service.MatchService
	Leaderboard *service.LeaderboardService
	Admin       *service.AdminService
	Logger      zerolog.Logger
}

// NewRouter 装配全局中间件与全部路由。
func NewRouter(deps Deps) *gin.Engine {
	r := gin.New()

	// 全局中间件：顺序为 RequestID -> Logger -> Recover。
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.Recover())

	// handler 聚合体。
	auth := &authHandler{player: deps.Player, admin: deps.Admin}
	players := &playerHandler{player: deps.Player, match: deps.Match}
	matches := &matchHandler{match: deps.Match, admin: deps.Admin}
	board := &leaderboardHandler{svc: deps.Leaderboard}

	// 鉴权中间件实例。
	playerAuth := middleware.PlayerAuth(deps.Player)
	adminAuth := middleware.AdminAuth(deps.Admin)

	api := r.Group("/api")
	{
		// 健康检查（无鉴权）。
		api.GET("/healthz", handleHealthz)

		// 鉴权（无需 PlayerAuth 的端点）。
		api.POST("/login", auth.handleLogin)
		api.POST("/logout", auth.handleLogout)
		api.POST("/admin/login", auth.handleAdminLogin)
		api.POST("/admin/logout", auth.handleAdminLogout)
		api.GET("/admin/status", auth.handleAdminStatus)

		// 需要玩家登录的端点。
		authed := api.Group("")
		authed.Use(playerAuth)
		{
			authed.GET("/me", auth.handleMe)

			authed.GET("/players", players.handleListPlayers)
			authed.GET("/players/:username", players.handleGetPlayer)
			authed.GET("/players/:username/history", players.handlePlayerHistory)
			authed.GET("/players/:username/matches", players.handlePlayerMatches)

			authed.POST("/matches", matches.handleRecordMatch)
			authed.GET("/matches", matches.handleListGlobalMatches)

			authed.GET("/leaderboard", board.handleLeaderboard)
			authed.GET("/compare", board.handleCompare)
		}

		// 需要管理员的端点。
		admin := api.Group("")
		admin.Use(adminAuth)
		{
			admin.DELETE("/matches/:id", matches.handleDeleteMatch)
			admin.GET("/admin/matches/deleted", matches.handleListDeletedMatches)
			admin.POST("/admin/matches/:id/restore", matches.handleRestoreMatch)
		}
	}

	return r
}
```

> 路由冲突说明：gin 的 `:username` 与 `:id` 都是同段通配，但它们分属不同 HTTP 方法与不同前缀路径（`GET /players/:username` vs `DELETE /matches/:id` vs `POST /admin/matches/:id/restore`），不会冲突。`/matches`（GET/POST）与 `/matches/:id`（DELETE）方法不同，gin 允许。

- [ ] 运行整包测试，确认 Task 5–10 全部转绿：

```
go test ./internal/handler/
```

预期输出（所有 handler 测试通过）：

```
ok  	go_ultra/internal/handler	0.xxxs
```

> 若个别 service 方法名与本计划接口（如 `GetByID`、`CreatePlayerSession`、`DeletePlayerSession`、`GetSession`、`ListDeleted`、`DeleteAdminSession`）不一致导致编译失败，按报错把对应方法在 service 包内补齐为契约要求的签名（这些都是 http 层明确需要的会话/便利方法），不要修改 handler 接口名。修齐后重跑本命令直至通过。

- [ ] 运行 race 检测（spec §9.3 要求）：

```
go test -race ./internal/handler/
```

预期输出：

```
ok  	go_ultra/internal/handler	0.xxxs
```

- [ ] commit：

```
git add server/internal/handler/router.go
git commit -m "feat(handler): add NewRouter wiring all middleware and routes"
```

---

### Task 12: config —— 配置加载（DB 路径，GO_ULTRA_DB 覆盖）

**Files:**
- Create: `server/internal/config/config.go`
- Test: `server/internal/config/config_test.go`

本 Task 实现配置加载：DB 路径默认 `./go_ultra.db`，可被环境变量 `GO_ULTRA_DB` 覆盖；监听地址固定 `:8080`。

步骤：

- [ ] 写失败测试 `server/internal/config/config_test.go`，内容完整如下：

```go
package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GO_ULTRA_DB", "")
	cfg := Load()
	if cfg.DBPath != "./go_ultra.db" {
		t.Fatalf("DBPath = %q, want ./go_ultra.db", cfg.DBPath)
	}
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
}

func TestLoad_DBOverride(t *testing.T) {
	t.Setenv("GO_ULTRA_DB", "/tmp/custom.db")
	cfg := Load()
	if cfg.DBPath != "/tmp/custom.db" {
		t.Fatalf("DBPath = %q, want /tmp/custom.db", cfg.DBPath)
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/config/
```

预期失败（编译错误，`Load`/`Config` 未定义）：

```
# go_ultra/internal/config [go_ultra/internal/config.test]
./config_test.go:...: undefined: Load
FAIL	go_ultra/internal/config [build failed]
```

- [ ] 写最小实现 `server/internal/config/config.go`，内容完整如下：

```go
package config

import "os"

// Config 持有运行期配置。
type Config struct {
	DBPath string // SQLite 文件路径
	Addr   string // HTTP 监听地址
}

// Load 从环境变量加载配置，提供默认值。
func Load() Config {
	dbPath := "./go_ultra.db"
	if v := os.Getenv("GO_ULTRA_DB"); v != "" {
		dbPath = v
	}
	return Config{
		DBPath: dbPath,
		Addr:   ":8080",
	}
}
```

- [ ] 运行确认通过：

```
go test ./internal/config/
```

预期输出：

```
ok  	go_ultra/internal/config	0.0xxs
```

- [ ] commit：

```
git add server/internal/config/config.go server/internal/config/config_test.go
git commit -m "feat(config): add config loader with GO_ULTRA_DB override"
```

---

### Task 13: main.go —— 程序装配 + 冒烟测试

**Files:**
- Create: `server/cmd/go_ultra/main.go`
- Create: `server/cmd/go_ultra/main_test.go`

本 Task 给出完整的程序装配：config → `db.New` → `sqlc.New` → 各 service → `AdminService.EnsurePassword(context.Background())`（若 `generated` 则打印到 stdout 并写 `logs/admin_password.txt`）→ `NewRouter` → 监听 `:8080`。全程用 `context` 而非 `nil`。为了让装配逻辑可被冒烟测试覆盖，把"构造 router"的部分抽成 `buildRouter(cfg) (*gin.Engine, func(), error)`，`main()` 只做编排与监听。

步骤：

- [ ] 写失败的冒烟测试 `server/cmd/go_ultra/main_test.go`，内容完整如下：

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"go_ultra/internal/config"
)

func TestBuildRouter_Healthz(t *testing.T) {
	cfg := config.Config{
		DBPath: filepath.Join(t.TempDir(), "smoke.db"),
		Addr:   ":0",
	}
	r, cleanup, err := buildRouter(cfg)
	if err != nil {
		t.Fatalf("buildRouter failed: %v", err)
	}
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("body = %q, want {\"status\":\"ok\"}", w.Body.String())
	}
}
```

- [ ] 运行确认失败：

```
go test ./cmd/go_ultra/
```

预期失败（编译错误，`buildRouter` 未定义）：

```
# go_ultra/cmd/go_ultra [go_ultra/cmd/go_ultra.test]
./main_test.go:...: undefined: buildRouter
FAIL	go_ultra/cmd/go_ultra [build failed]
```

- [ ] 写最小实现 `server/cmd/go_ultra/main.go`，内容完整如下：

```go
package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"go_ultra/internal/config"
	"go_ultra/internal/db"
	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/handler"
	"go_ultra/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// newLogger 创建一个写到 stdout 的结构化 logger。
func newLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// buildRouter 完成全部装配并返回 router、清理函数与错误。
// 全程使用 context.Background()，不传 nil。
func buildRouter(cfg config.Config) (*gin.Engine, func(), error) {
	ctx := context.Background()
	logger := newLogger()

	sqlDB, err := db.New(cfg.DBPath)
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() { _ = sqlDB.Close() }

	q := sqlc.New(sqlDB)

	playerSvc := service.NewPlayerService(q, sqlDB)
	matchSvc := service.NewMatchService(q, sqlDB)
	leaderboardSvc := service.NewLeaderboardService(q, sqlDB)
	adminSvc := service.NewAdminService(q, sqlDB)

	plaintext, generated, err := adminSvc.EnsurePassword(ctx)
	if err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if generated {
		// 首次启动：打印明文并落盘，供管理员登录。
		logger.Info().Msg("generated initial admin password (see logs/admin_password.txt)")
		// 直接写 stdout，避免被日志 JSON 包裹，便于复制。
		os.Stdout.WriteString("\n===========================================\n")
		os.Stdout.WriteString("go_ultra admin password (first start only):\n")
		os.Stdout.WriteString(plaintext + "\n")
		os.Stdout.WriteString("===========================================\n\n")

		if err := writeAdminPassword(plaintext); err != nil {
			logger.Error().Err(err).Msg("failed to write logs/admin_password.txt")
		}
	}

	deps := handler.Deps{
		Player:      playerSvc,
		Match:       matchSvc,
		Leaderboard: leaderboardSvc,
		Admin:       adminSvc,
		Logger:      logger,
	}
	return handler.NewRouter(deps), cleanup, nil
}

// writeAdminPassword 把首启明文密码写入 logs/admin_password.txt。
func writeAdminPassword(plaintext string) error {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return err
	}
	path := filepath.Join("logs", "admin_password.txt")
	content := "go_ultra admin password (generated " +
		time.Now().UTC().Format(time.RFC3339) + "):\n" + plaintext + "\n"
	return os.WriteFile(path, []byte(content), 0o600)
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	cfg := config.Load()
	logger := newLogger()

	router, cleanup, err := buildRouter(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to build router")
	}
	defer cleanup()

	logger.Info().Str("addr", cfg.Addr).Msg("go_ultra listening")
	if err := router.Run(cfg.Addr); err != nil {
		logger.Fatal().Err(err).Msg("server stopped")
	}
}
```

- [ ] 运行确认通过：

```
go test ./cmd/go_ultra/
```

预期输出：

```
ok  	go_ultra/cmd/go_ultra	0.xxxs
```

- [ ] commit：

```
git add server/cmd/go_ultra/main.go server/cmd/go_ultra/main_test.go
git commit -m "feat(cmd): wire config/db/services/router in main with admin password bootstrap"
```

---

### Task 14: CSRF Origin 校验中间件

本 Task 补齐 spec §8.1 安全表中 CSRF 一项的"校验 `Origin` 头"部分（`SameSite=Lax` 已由 session 层 cookie 属性实现，详见 session 层 Task）。实现一个 Gin 中间件 `OriginCheck`，对所有非安全 HTTP 方法（`POST`/`PUT`/`PATCH`/`DELETE`）校验请求的 `Origin` 头是否落在白名单内，安全方法（`GET`/`HEAD`/`OPTIONS`）一律放行。

**策略说明（必须理解后再实现）**：

- 现代浏览器对**所有跨站写请求**（含 form 提交与 `fetch`/`XHR`）都会自动带上 `Origin` 头，且该头不可被页面脚本伪造。因此"`Origin` 缺失"在一个仅服务浏览器客户端的写端点上属于**可疑**情形。本中间件对非安全方法采取**缺失即拒绝**（fail-closed）策略：`Origin` 为空时直接 `abort(c, domain.ErrInvalidParam)`，而不是放行。这与 §8.1"所有 POST/DELETE 校验 Origin 头"的字面要求一致——"校验"意味着没有合法 Origin 就不通过。
- 代价：少数非浏览器客户端（curl、健康探针等）若用非安全方法访问会被拒。本系统的写端点（`/api/login`、`/api/matches`、`/api/admin/*`）全部面向浏览器，健康检查 `GET /api/healthz` 是安全方法不受影响，故该代价可接受。
- 白名单匹配为**精确字符串相等**（scheme + host + port 整体），不做子串或后缀匹配，避免 `evil-cloudflare.com` 之类的绕过。

**Files:**
- Create: `server/internal/middleware/csrf.go`
- Test: `server/internal/middleware/csrf_test.go`
- Edit: `server/internal/config/config.go`（新增 `AllowedOrigins` 字段与默认值）
- Edit: `server/internal/handler/router.go`（`Deps` 新增 `AllowedOrigins`，`/api` 组挂载 `OriginCheck`）
- Edit: `server/cmd/go_ultra/main.go`（`buildRouter` 把 `cfg.AllowedOrigins` 填入 `Deps`）

步骤：

- [ ] 写失败测试 `server/internal/middleware/csrf_test.go`，内容完整如下：

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newOriginTestEngine 构造一个最小 gin engine：根中间件挂 OriginCheck，
// 注册一个对所有方法都返回 200 的探针路由，便于断言"放行 vs 拒绝"。
func newOriginTestEngine(allowed []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OriginCheck(allowed))
	handler := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	r.GET("/probe", handler)
	r.POST("/probe", handler)
	r.DELETE("/probe", handler)
	return r
}

func TestOriginCheck(t *testing.T) {
	const allowedOrigin = "https://go-ultra.example.com"
	allowed := []string{allowedOrigin, "http://localhost:5173"}

	tests := []struct {
		name       string
		method     string
		setOrigin  bool
		origin     string
		wantStatus int
	}{
		{
			name:       "safe GET passes without origin",
			method:     http.MethodGet,
			setOrigin:  false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "safe GET passes with foreign origin",
			method:     http.MethodGet,
			setOrigin:  true,
			origin:     "https://evil.example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with allowed origin passes",
			method:     http.MethodPost,
			setOrigin:  true,
			origin:     allowedOrigin,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with dev origin passes",
			method:     http.MethodPost,
			setOrigin:  true,
			origin:     "http://localhost:5173",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with foreign origin rejected",
			method:     http.MethodPost,
			setOrigin:  true,
			origin:     "https://evil.example.com",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST with missing origin rejected",
			method:     http.MethodPost,
			setOrigin:  false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DELETE with foreign origin rejected",
			method:     http.MethodDelete,
			setOrigin:  true,
			origin:     "https://evil.example.com",
			wantStatus: http.StatusBadRequest,
		},
	}

	r := newOriginTestEngine(allowed)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/probe", nil)
			if tc.setOrigin {
				req.Header.Set("Origin", tc.origin)
			}
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
			// 被拒时应返回统一错误格式，code 为 INVALID_PARAM。
			if tc.wantStatus == http.StatusBadRequest {
				if body := w.Body.String(); body != `{"error":{"code":"INVALID_PARAM","message":"参数无效"}}` {
					t.Fatalf("error body = %q, want INVALID_PARAM envelope", body)
				}
			}
		})
	}
}
```

- [ ] 运行确认失败：

```
go test ./internal/middleware/
```

预期失败（编译错误，`OriginCheck` 未定义）：

```
# go_ultra/internal/middleware [go_ultra/internal/middleware.test]
./csrf_test.go:...: undefined: OriginCheck
FAIL	go_ultra/internal/middleware [build failed]
```

- [ ] 写最小实现 `server/internal/middleware/csrf.go`，内容完整如下：

```go
package middleware

import (
	"net/http"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
)

// safeMethods 是无副作用、无需 CSRF 防护的 HTTP 方法。
var safeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// OriginCheck 对非安全方法（POST/PUT/PATCH/DELETE）校验 Origin 头。
//
// 策略（fail-closed）：
//   - 安全方法（GET/HEAD/OPTIONS）直接放行，不读 Origin。
//   - 非安全方法：Origin 头缺失视为可疑（浏览器对跨站写请求必带 Origin），拒绝。
//   - 非安全方法：Origin 须精确等于 allowedOrigins 中的某一项，否则拒绝。
//
// 拒绝时统一返回 domain.ErrInvalidParam（400 / INVALID_PARAM）。
func OriginCheck(allowedOrigins []string) gin.HandlerFunc {
	// 预构建集合，O(1) 查找，避免每请求线性扫描。
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(c *gin.Context) {
		if safeMethods[c.Request.Method] {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			abort(c, domain.ErrInvalidParam)
			return
		}
		if !allowed[origin] {
			abort(c, domain.ErrInvalidParam)
			return
		}

		c.Next()
	}
}
```

> 说明：`abort` 是本包内 `auth.go` 已定义的非导出 helper（把 `*domain.Error` 写成 `{"error":{"code","message"}}`），此处直接复用，保证错误响应格式与其它中间件一致。

- [ ] 运行确认通过：

```
go test ./internal/middleware/
```

预期输出：

```
ok  	go_ultra/internal/middleware	0.0xxs
```

- [ ] 修改 `server/internal/config/config.go`，给 `Config` 增加 `AllowedOrigins` 字段。把结构体定义：

```go
// Config 持有运行期配置。
type Config struct {
	DBPath string // SQLite 文件路径
	Addr   string // HTTP 监听地址
}
```

改为：

```go
// Config 持有运行期配置。
type Config struct {
	DBPath         string   // SQLite 文件路径
	Addr           string   // HTTP 监听地址
	AllowedOrigins []string // CSRF Origin 头白名单（精确匹配）
}
```

- [ ] 修改同文件 `Load()`，填充 `AllowedOrigins` 默认值。把：

```go
	return Config{
		DBPath: dbPath,
		Addr:   ":8080",
	}
```

改为：

```go
	return Config{
		DBPath: dbPath,
		Addr:   ":8080",
		// 开发期 Vite dev server 源；生产部署时追加 Cloudflare Tunnel 域名
		// （如 "https://go-ultra.example.com"）。
		AllowedOrigins: []string{"http://localhost:5173"},
	}
```

> 说明：生产 Cloudflare 域名因人而异，不写死在默认值里；部署者在此切片追加自己的 `https://<域名>` 即可（README 部署章节会引导）。开发期同源由 Vite proxy 代理，浏览器实际发出的写请求 Origin 为 `http://localhost:5173`，故默认含之。

- [ ] 在 `server/internal/config/config_test.go` 末尾追加一个断言默认白名单的测试：

```go

func TestLoad_AllowedOriginsDefault(t *testing.T) {
	t.Setenv("GO_ULTRA_DB", "")
	cfg := Load()
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("AllowedOrigins = %v, want [http://localhost:5173]", cfg.AllowedOrigins)
	}
}
```

- [ ] 运行确认 config 包通过：

```
go test ./internal/config/
```

预期输出：

```
ok  	go_ultra/internal/config	0.0xxs
```

- [ ] 修改 `server/internal/handler/router.go`，给 `Deps` 增加 `AllowedOrigins` 字段。把：

```go
// Deps 是装配 router 所需的全部依赖。
type Deps struct {
	Player      *service.PlayerService
	Match       *service.MatchService
	Leaderboard *service.LeaderboardService
	Admin       *service.AdminService
	Logger      zerolog.Logger
}
```

改为：

```go
// Deps 是装配 router 所需的全部依赖。
type Deps struct {
	Player         *service.PlayerService
	Match          *service.MatchService
	Leaderboard    *service.LeaderboardService
	Admin          *service.AdminService
	Logger         zerolog.Logger
	AllowedOrigins []string // CSRF Origin 头白名单，传给 middleware.OriginCheck
}
```

- [ ] 在同文件 `NewRouter` 内，把 `OriginCheck` 挂到 `/api` 组上（它对安全方法自动放行，故覆盖所有写端点而不影响 `GET /api/healthz` 等）。把：

```go
	api := r.Group("/api")
	{
		// 健康检查（无鉴权）。
		api.GET("/healthz", handleHealthz)
```

改为：

```go
	api := r.Group("/api")
	// CSRF 防护：对所有非安全方法（POST/PUT/PATCH/DELETE）校验 Origin 头。
	// 安全方法（GET/HEAD/OPTIONS）由中间件内部放行，故 /api/healthz 不受影响。
	api.Use(middleware.OriginCheck(deps.AllowedOrigins))
	{
		// 健康检查（无鉴权）。
		api.GET("/healthz", handleHealthz)
```

- [ ] 修改 `server/cmd/go_ultra/main.go` 的 `buildRouter`，把 `cfg.AllowedOrigins` 填进 `Deps`。把：

```go
	deps := handler.Deps{
		Player:      playerSvc,
		Match:       matchSvc,
		Leaderboard: leaderboardSvc,
		Admin:       adminSvc,
		Logger:      logger,
	}
```

改为：

```go
	deps := handler.Deps{
		Player:         playerSvc,
		Match:          matchSvc,
		Leaderboard:    leaderboardSvc,
		Admin:          adminSvc,
		Logger:         logger,
		AllowedOrigins: cfg.AllowedOrigins,
	}
```

- [ ] 运行受影响的三个包，确认整体编译通过且测试转绿：

```
go test ./internal/middleware/ ./internal/config/ ./internal/handler/ ./cmd/go_ultra/
```

预期输出：

```
ok  	go_ultra/internal/middleware	0.0xxs
ok  	go_ultra/internal/config	0.0xxs
ok  	go_ultra/internal/handler	0.xxxs
ok  	go_ultra/cmd/go_ultra	0.xxxs
```

> 说明：`cmd/go_ultra` 的冒烟测试构造 `config.Config{...}` 字面量时未设 `AllowedOrigins`（零值 `nil`），`OriginCheck(nil)` 对 `GET /api/healthz` 仍放行，故 `TestBuildRouter_Healthz` 不受影响、继续通过。

- [ ] commit：

```
git add server/internal/middleware/csrf.go server/internal/middleware/csrf_test.go server/internal/config/config.go server/internal/config/config_test.go server/internal/handler/router.go server/cmd/go_ultra/main.go
git commit -m "feat(middleware): add Origin header CSRF check for unsafe methods"
```

---

### Task 15: 管理员登录失败指数退避（domain 哨兵 ErrRateLimited）

**Files:**
- Modify: `server/internal/domain/errors.go`
- Test: `server/internal/domain/errors_test.go`

本任务在 domain 层补上 spec §6.2 规定的 429 错误。现有 `errors.go` 已有 `*Error` 类型（字段 `Code/Message/Status/Cause`）与一组哨兵（`ErrInvalidParam` 等），但**尚无** `ErrRateLimited`。前端在登录流程里已预期收到 `RATE_LIMITED`，后端必须能产出它。本任务只新增一个哨兵变量并断言其字段，不引入任何行为。

- [ ] 写失败测试。在 `server/internal/domain/errors_test.go` 中追加一个表驱动测试，断言 `ErrRateLimited` 的 `Code`、`Status`、`Message` 与 spec §6.2 一致。若该文件已存在，追加下面的测试函数；若不存在，创建该文件并写入完整内容：

```go
package domain

import "testing"

func TestErrRateLimited(t *testing.T) {
	if ErrRateLimited == nil {
		t.Fatal("ErrRateLimited must be defined")
	}
	if got := ErrRateLimited.Code; got != "RATE_LIMITED" {
		t.Errorf("Code = %q, want %q", got, "RATE_LIMITED")
	}
	if got := ErrRateLimited.Status; got != 429 {
		t.Errorf("Status = %d, want %d", got, 429)
	}
	if ErrRateLimited.Message == "" {
		t.Error("Message must not be empty")
	}
	// 哨兵必须实现 error 接口且 Error() 含 Code，便于日志定位
	if got := ErrRateLimited.Error(); got == "" {
		t.Error("Error() must not be empty")
	}
}
```

- [ ] 运行确认失败。命令：

```bash
cd server && go test ./internal/domain/ -run TestErrRateLimited
```

预期失败：编译错误 `undefined: ErrRateLimited`（因为哨兵尚未定义，测试包无法编译）。

- [ ] 写最小实现。在 `server/internal/domain/errors.go` 的预定义哨兵块末尾（紧跟 `ErrInternal` 之后）新增一行：

```go
var ErrRateLimited     = &Error{Code: "RATE_LIMITED",     Message: "尝试过于频繁，请稍后", Status: 429}
```

新增后该块应形如：

```go
var ErrInvalidBody      = &Error{Code: "INVALID_BODY",      Message: "请求体无效",       Status: 400}
var ErrInvalidParam     = &Error{Code: "INVALID_PARAM",     Message: "参数无效",         Status: 400}
var ErrInternal         = &Error{Code: "INTERNAL",          Message: "服务器内部错误",   Status: 500}
var ErrRateLimited      = &Error{Code: "RATE_LIMITED",      Message: "尝试过于频繁，请稍后", Status: 429}
```

- [ ] 运行确认通过。命令：

```bash
cd server && go test ./internal/domain/ -run TestErrRateLimited -v
```

预期输出包含 `--- PASS: TestErrRateLimited` 与结尾 `ok  	go_ultra/internal/domain`。

- [ ] commit：

```bash
git add server/internal/domain/errors.go server/internal/domain/errors_test.go
git commit -m "feat(domain): add ErrRateLimited sentinel for 429 responses"
```

---

### Task 16: 管理员登录失败指数退避（service + handler）

**Files:**
- Modify: `server/internal/service/admin.go`
- Test: `server/internal/service/admin_lockout_test.go`
- Modify: `server/internal/handler/auth.go`
- Test: `server/internal/handler/auth_lockout_test.go`

本任务实现 spec §8.1 的"管理员密码暴力穷举"缓解：错误密码指数退避，失败次数 `N` → 锁定 `2^N` 秒，封顶 1 小时。

**设计权衡（必须写进代码注释，便于后续维护者理解）：**

1. **按全局锁定，不按 IP。** 本系统只有一个管理员（单一密码），因此用一个全局计数器与一个全局锁定截止时间即可。**故意不按 IP 区分**——若按 IP，攻击者轮换源 IP（代理池/僵尸网络）即可绕过退避；全局锁定让任何来源的连续失败共同累积，从根上堵死穷举。代价是：攻击者可借此对管理员制造短时"拒登"（最多 1 小时），对单管理员的朋友圈场景可接受。
2. **状态存内存，不落库。** 用 `AdminService` 上的 `sync.Mutex` + 两个字段保存计数与锁定截止。进程重启后清零——对单机朋友圈部署可接受，且重启本身需要本机访问权限，不构成穷举通道。代价是：重启可重置退避；同样在威胁模型内可接受。
3. **`nowFunc` 可注入以便测试。** 新增 `nowFunc func() time.Time` 字段，零值时回退到 `time.Now`，测试可注入假时钟，**无需任何 `time.Sleep`**。

`AdminService` 现有定义为 `AdminService{q *sqlc.Queries; db *sql.DB}`，构造函数 `NewAdminService(q, db)`。本任务给结构体**追加**字段（零值即可用，构造函数签名不变），并新增三个方法。

- [ ] 写失败的 service 层测试。创建 `server/internal/service/admin_lockout_test.go`，内容如下。测试直接注入假时钟、操作公开方法，断言：连续失败后 `CheckLockout` 返回 `domain.ErrRateLimited`；`2^N` 公式与 1 小时封顶；时间推进越过锁定后自动放行；`ResetLoginFailures` 立即解锁并把退避归零：

```go
package service

import (
	"errors"
	"testing"
	"time"

	"go_ultra/internal/domain"
)

// newLockoutTestService 构造一个仅用于退避逻辑测试的 AdminService，
// 不触碰 DB（退避状态全在内存），并注入可控时钟。
func newLockoutTestService(now *time.Time) *AdminService {
	s := &AdminService{}
	s.nowFunc = func() time.Time { return *now }
	return s
}

func TestCheckLockout_NoFailures(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	if err := s.CheckLockout(); err != nil {
		t.Fatalf("CheckLockout with no failures: got %v, want nil", err)
	}
}

func TestRecordLoginFailure_LocksWithExponentialBackoff(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	// 第 1 次失败 → 锁定 2^1 = 2 秒。
	s.RecordLoginFailure()
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("after 1 failure: got %v, want ErrRateLimited", err)
	}

	// 推进 1 秒（仍在 2 秒锁定内）→ 仍锁定。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("at +1s of 2s lock: got %v, want ErrRateLimited", err)
	}

	// 推进到第 2 秒末（>= 锁定截止）→ 放行。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("at +2s of 2s lock: got %v, want nil", err)
	}
}

func TestRecordLoginFailure_BackoffGrows(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	// 累计 3 次失败 → 锁定 2^3 = 8 秒。
	s.RecordLoginFailure()
	s.RecordLoginFailure()
	s.RecordLoginFailure()

	// +7 秒仍锁定。
	now = now.Add(7 * time.Second)
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("at +7s of 8s lock: got %v, want ErrRateLimited", err)
	}

	// +8 秒放行。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("at +8s of 8s lock: got %v, want nil", err)
	}
}

func TestRecordLoginFailure_CapsAtOneHour(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	// 失败 20 次：2^20 秒远超 1 小时，必须封顶到 3600 秒。
	for i := 0; i < 20; i++ {
		s.RecordLoginFailure()
	}

	// +3599 秒仍锁定。
	now = now.Add(3599 * time.Second)
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("at +3599s of capped lock: got %v, want ErrRateLimited", err)
	}

	// +3600 秒放行（证明封顶为 1 小时，未无限增长）。
	now = now.Add(1 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("at +3600s of capped lock: got %v, want nil", err)
	}
}

func TestResetLoginFailures_Unlocks(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s := newLockoutTestService(&now)

	s.RecordLoginFailure()
	s.RecordLoginFailure()
	if err := s.CheckLockout(); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("after 2 failures: got %v, want ErrRateLimited", err)
	}

	// 重置后立即放行。
	s.ResetLoginFailures()
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("after reset: got %v, want nil", err)
	}

	// 重置还必须把计数归零：重置后第 1 次失败应只锁 2^1 = 2 秒，而非延续之前的指数。
	s.RecordLoginFailure()
	now = now.Add(2 * time.Second)
	if err := s.CheckLockout(); err != nil {
		t.Fatalf("first failure after reset should lock only 2s; at +2s got %v, want nil", err)
	}
}
```

- [ ] 运行确认失败。命令：

```bash
cd server && go test ./internal/service/ -run "TestCheckLockout|TestRecordLoginFailure|TestResetLoginFailures"
```

预期失败：编译错误，类似 `s.nowFunc undefined (type *AdminService has no field or method nowFunc)` 以及 `s.CheckLockout undefined` / `s.RecordLoginFailure undefined` / `s.ResetLoginFailures undefined`。

- [ ] 写最小实现（service）。在 `server/internal/service/admin.go` 顶部 import 块补充 `sync` 与 `time`（若已存在则不重复），把 `AdminService` 结构体替换为下面带退避字段的版本，并在文件末尾追加三个方法。

将结构体定义：

```go
type AdminService struct {
	q  *sqlc.Queries
	db *sql.DB
}
```

替换为：

```go
type AdminService struct {
	q  *sqlc.Queries
	db *sql.DB

	// 登录失败指数退避状态。设计取舍：
	//  1. 全局锁定（非按 IP）——系统仅一个管理员，全局计数让攻击者无法靠轮换 IP 绕过退避。
	//  2. 状态存内存，进程重启清零——单机朋友圈部署可接受；重启需本机访问权限，不构成穷举通道。
	//  3. nowFunc 可注入以便测试，零值回退 time.Now，避免测试依赖真实时钟与 sleep。
	mu          sync.Mutex
	failCount   int
	lockedUntil time.Time
	nowFunc     func() time.Time
}
```

在文件末尾追加：

```go
// adminLockoutCap 是单次锁定时长的封顶（spec §8.1：封顶 1 小时）。
const adminLockoutCap = time.Hour

// now 返回当前时间，优先用可注入的 nowFunc（测试用），否则回退 time.Now。
// 调用方必须已持有 s.mu。
func (s *AdminService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
}

// CheckLockout 在当前处于锁定窗口内时返回 domain.ErrRateLimited，否则返回 nil。
// 应在校验管理员密码之前调用。
func (s *AdminService) CheckLockout() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.now().Before(s.lockedUntil) {
		return domain.ErrRateLimited
	}
	return nil
}

// RecordLoginFailure 记录一次密码校验失败：失败次数自增，并把锁定截止时间
// 设为 now + min(2^failCount 秒, 1 小时)。
func (s *AdminService) RecordLoginFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCount++
	backoff := time.Duration(1<<uint(s.failCount)) * time.Second
	if backoff > adminLockoutCap || backoff <= 0 { // backoff<=0 防大位移溢出
		backoff = adminLockoutCap
	}
	s.lockedUntil = s.now().Add(backoff)
}

// ResetLoginFailures 在登录成功后清空失败计数与锁定窗口。
func (s *AdminService) ResetLoginFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failCount = 0
	s.lockedUntil = time.Time{}
}
```

注意：`RecordLoginFailure` 用 `1<<uint(s.failCount)`，当 `failCount` 较大时左移会溢出成 0 或负值，`backoff <= 0` 分支把它收敛到封顶，保证测试中"失败 20 次"仍是 1 小时封顶而非异常值。

- [ ] 运行确认通过（service）。命令：

```bash
cd server && go test ./internal/service/ -run "TestCheckLockout|TestRecordLoginFailure|TestResetLoginFailures" -v
```

预期输出每个用例一行 `--- PASS`，结尾 `ok  	go_ultra/internal/service`。

- [ ] 写失败的 handler 层测试。创建 `server/internal/handler/auth_lockout_test.go`，内容如下。它直接调用 `handleAdminLogin` 的封装路由，用一个实现新接口 `authAdminService` 的 fake（含退避三方法），断言：锁定期间返回 `429` 且 body 为 `{"error":{"code":"RATE_LIMITED",...}}`；密码错误时触发 `RecordLoginFailure` 并返回 `400 INVALID_PARAM`；密码正确时触发 `ResetLoginFailures`。

```go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go_ultra/internal/domain"
)

// fakeAdminAuth 实现 authAdminService，用于在不依赖 DB 的情况下测试退避分支。
type fakeAdminAuth struct {
	locked       bool
	pwOK         bool
	recordCalled int
	resetCalled  int
}

func (f *fakeAdminAuth) VerifyPassword(_ ginCtxContext, _ string) (bool, error) {
	return f.pwOK, nil
}
func (f *fakeAdminAuth) CreateAdminSession(_ ginCtxContext) (string, time.Time, error) {
	return "tok", time.Now().Add(30 * time.Minute), nil
}
func (f *fakeAdminAuth) CheckAdminSession(_ ginCtxContext, _ string) (bool, time.Time, error) {
	return true, time.Now().Add(30 * time.Minute), nil
}
func (f *fakeAdminAuth) DeleteAdminSession(_ ginCtxContext, _ string) error { return nil }
func (f *fakeAdminAuth) CheckLockout() error {
	if f.locked {
		return domain.ErrRateLimited
	}
	return nil
}
func (f *fakeAdminAuth) RecordLoginFailure() { f.recordCalled++ }
func (f *fakeAdminAuth) ResetLoginFailures() { f.resetCalled++ }

// doAdminLogin 构造一个仅挂载 admin login 路由的引擎并发起请求。
func doAdminLogin(t *testing.T, svc authAdminService, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/admin/login", func(c *gin.Context) {
		handleAdminLogin(c, svc)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeErrCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return resp.Error.Code
}

func TestHandleAdminLogin_LockedReturns429(t *testing.T) {
	svc := &fakeAdminAuth{locked: true}
	w := doAdminLogin(t, svc, `{"password":"whatever"}`)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if code := decodeErrCode(t, w); code != "RATE_LIMITED" {
		t.Fatalf("error.code = %q, want RATE_LIMITED", code)
	}
	if svc.recordCalled != 0 {
		t.Errorf("RecordLoginFailure called %d times while locked, want 0", svc.recordCalled)
	}
}

func TestHandleAdminLogin_WrongPasswordRecordsFailure(t *testing.T) {
	svc := &fakeAdminAuth{locked: false, pwOK: false}
	w := doAdminLogin(t, svc, `{"password":"wrong"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if code := decodeErrCode(t, w); code != "INVALID_PARAM" {
		t.Fatalf("error.code = %q, want INVALID_PARAM", code)
	}
	if svc.recordCalled != 1 {
		t.Errorf("RecordLoginFailure called %d times, want 1", svc.recordCalled)
	}
	if svc.resetCalled != 0 {
		t.Errorf("ResetLoginFailures called %d times on failure, want 0", svc.resetCalled)
	}
}

func TestHandleAdminLogin_SuccessResetsFailures(t *testing.T) {
	svc := &fakeAdminAuth{locked: false, pwOK: true}
	w := doAdminLogin(t, svc, `{"password":"correct"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if svc.resetCalled != 1 {
		t.Errorf("ResetLoginFailures called %d times, want 1", svc.resetCalled)
	}
	if svc.recordCalled != 0 {
		t.Errorf("RecordLoginFailure called %d times on success, want 0", svc.recordCalled)
	}
}
```

说明：测试用 `ginCtxContext` 作为 `VerifyPassword` 等方法的第一个参数类型别名占位——它必须与 `auth.go` 中 `authAdminService` 接口实际使用的上下文类型一致。下一步实现里 `authAdminService` 用的是 `context.Context`，因此在 `auth_lockout_test.go` 顶部 import 区下方加一行类型别名，使 fake 的签名与接口对齐：

```go
type ginCtxContext = contextContext
```

为避免再引入间接别名，**直接把 fake 方法签名里的 `ginCtxContext` 全部改为 `context.Context`，并在 import 中加入 `"context"`，删除上面这条别名说明**。即 fake 的方法写成：

```go
func (f *fakeAdminAuth) VerifyPassword(_ context.Context, _ string) (bool, error) { return f.pwOK, nil }
func (f *fakeAdminAuth) CreateAdminSession(_ context.Context) (string, time.Time, error) {
	return "tok", time.Now().Add(30 * time.Minute), nil
}
func (f *fakeAdminAuth) CheckAdminSession(_ context.Context, _ string) (bool, time.Time, error) {
	return true, time.Now().Add(30 * time.Minute), nil
}
func (f *fakeAdminAuth) DeleteAdminSession(_ context.Context, _ string) error { return nil }
```

并把测试文件 import 区写为：

```go
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go_ultra/internal/domain"
)
```

（最终文件中不应出现 `ginCtxContext`／`contextContext` 任何字样——以这一步的 `context.Context` 版本为准。）

- [ ] 运行确认失败。命令：

```bash
cd server && go test ./internal/handler/ -run "TestHandleAdminLogin_Locked|TestHandleAdminLogin_WrongPassword|TestHandleAdminLogin_SuccessResets"
```

预期失败：编译错误，`authAdminService` 接口尚无 `CheckLockout`／`RecordLoginFailure`／`ResetLoginFailures`，fake 因实现了这些方法但接口未声明而无法用作 `authAdminService`（`fakeAdminAuth does not implement authAdminService` 或接口断言不匹配），且 `handleAdminLogin` 尚未调用退避方法。

- [ ] 写最小实现（handler 接口）。在 `server/internal/handler/auth.go` 的 `authAdminService` 接口定义中，于现有 `VerifyPassword/CreateAdminSession/CheckAdminSession/DeleteAdminSession` 之后追加三个退避方法。把接口：

```go
type authAdminService interface {
	VerifyPassword(ctx context.Context, pw string) (bool, error)
	CreateAdminSession(ctx context.Context) (token string, expiresAt time.Time, err error)
	CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error)
	DeleteAdminSession(ctx context.Context, token string) error
}
```

替换为：

```go
type authAdminService interface {
	VerifyPassword(ctx context.Context, pw string) (bool, error)
	CreateAdminSession(ctx context.Context) (token string, expiresAt time.Time, err error)
	CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error)
	DeleteAdminSession(ctx context.Context, token string) error
	CheckLockout() error
	RecordLoginFailure()
	ResetLoginFailures()
}
```

- [ ] 写最小实现（handleAdminLogin）。把 `server/internal/handler/auth.go` 中现有的 `handleAdminLogin` 方法完整替换为下面版本（保持阶段 4 的方法接收者 `(h *authHandler)` 与 `respondError` 风格不变）。它在校验前先 `CheckLockout()`，锁定即返回 429；密码错误先 `RecordLoginFailure()` 再返回 `ErrInvalidParam`；成功先 `ResetLoginFailures()` 再建会话、设 cookie、返回 200 `{expires_at}`：

```go
// handleAdminLogin 校验密码并创建管理员会话；带登录失败指数退避。
func (h *authHandler) handleAdminLogin(c *gin.Context) {
	// 进入即检查退避锁定：锁定期间直接返回 429 RATE_LIMITED，
	// 不消耗 bcrypt 校验，也不再累加失败计数。
	if err := h.admin.CheckLockout(); err != nil {
		respondError(c, err)
		return
	}

	var req adminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.ErrInvalidBody.WithCause(err))
		return
	}

	ok, err := h.admin.VerifyPassword(c.Request.Context(), req.Password)
	if err != nil {
		respondError(c, err)
		return
	}
	if !ok {
		// 密码错误：累加失败计数（触发/延长指数退避），返回 400。
		h.admin.RecordLoginFailure()
		respondError(c, domain.ErrInvalidParam)
		return
	}

	// 密码正确：清空退避状态，再建立管理员会话。
	h.admin.ResetLoginFailures()

	token, expiresAt, err := h.admin.CreateAdminSession(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	setAdminCookie(c, token, time.Until(expiresAt))
	c.JSON(http.StatusOK, gin.H{"expires_at": expiresAt.UTC().Format(time.RFC3339)})
}
```

本函数复用阶段 4 已有的 `adminLoginRequest` 类型、`setAdminCookie` helper 与 `respondError`（把 `*domain.Error` 写成 `{"error":{"code","message"}}` 并设对应 HTTP 状态——`ErrRateLimited` 因此自动产出 `429 RATE_LIMITED`）。无需新增 import（`net/http`、`time`、`go_ultra/internal/domain` 阶段 4 已引入）。

- [ ] 运行确认通过（handler）。命令：

```bash
cd server && go test ./internal/handler/ -run "TestHandleAdminLogin_Locked|TestHandleAdminLogin_WrongPassword|TestHandleAdminLogin_SuccessResets" -v
```

预期输出三个用例均 `--- PASS`，结尾 `ok  	go_ultra/internal/handler`。

- [ ] 跑全量后端测试（含竞态检测，验证内存退避状态在并发下安全）。命令：

```bash
cd server && go test -race ./...
```

预期：全部 `ok`，无 `DATA RACE` 报告。

- [ ] commit：

```bash
git add server/internal/service/admin.go server/internal/service/admin_lockout_test.go server/internal/handler/auth.go server/internal/handler/auth_lockout_test.go
git commit -m "feat(service): add exponential backoff for admin login brute-force"
```

---


### Task 17: 全量验证 —— 整仓测试、race、覆盖率

**Files:**
- 无新增文件（仅运行验证命令）

本 Task 跑通整个后端测试套件，确认阶段 4 与前序阶段共同通过，覆盖率达到 spec §9.4 对 handler 层（≥ 70%）的目标。

步骤：

- [ ] 运行整仓测试：

```
go test ./...
```

预期输出（所有包通过）：

```
ok  	go_ultra/internal/config	0.0xxs
ok  	go_ultra/internal/domain	0.0xxs
ok  	go_ultra/internal/handler	0.xxxs
ok  	go_ultra/internal/middleware	0.0xxs
ok  	go_ultra/internal/session	0.0xxs
ok  	go_ultra/internal/service	0.xxxs
ok  	go_ultra/cmd/go_ultra	0.xxxs
...
```

- [ ] 运行 race 检测：

```
go test -race ./...
```

预期输出：所有包 `ok`，无 `DATA RACE` 报告。

- [ ] 运行 handler 层覆盖率，确认 ≥ 70%：

```
go test -cover ./internal/handler/ ./internal/middleware/ ./internal/session/
```

预期输出类似（coverage 数字 ≥ 70%）：

```
ok  	go_ultra/internal/handler	0.xxxs	coverage: 8x.x% of statements
ok  	go_ultra/internal/middleware	0.0xxs	coverage: 9x.x% of statements
ok  	go_ultra/internal/session	0.0xxs	coverage: 100.0% of statements
```

- [ ] 运行 `go vet` 静态检查：

```
go vet ./...
```

预期输出：无任何报告（命令静默成功）。

- [ ] commit（记录验证里程碑；若工作树无改动则跳过本步）：

```
git add -A
git commit -m "chore(http): phase 4 http layer and assembly fully green (tests/race/vet)" --allow-empty
```

---

至此，阶段 4（http 层 + 程序装配）完成：session 层、5 个中间件、统一错误响应、6 个 handler 文件、router 装配、config、main 装配与冒烟测试均已按契约实现并通过测试。后续阶段（前端）可基于此后端的稳定 API 契约进行开发。

---

实施计划"阶段 4"部分已完整给出（14 个 Task，从 session 层到 main.go 装配，每个 Task 含完整可粘贴代码、TDD 步骤与 commit）。相关文件路径（仓库根 `go_ultra/` 为基准）：
- `server/internal/session/session.go`、`server/internal/handler/{response,health,auth,player,match,leaderboard,router}.go`、`server/internal/middleware/{middleware,auth}.go`、`server/internal/config/config.go`、`server/cmd/go_ultra/main.go` 及各自测试文件。

設計与契約遵守要点：统一错误响应 `{"error":{"code,message}}` 从 `*domain.Error` 取值、非 domain 错误兜底 `ErrInternal` 并 log Cause；`NewRouter(Deps)` 签名与 `Deps` 字段严格按契约；cookie 名 `go_ultra_session`/`go_ultra_admin`、TTL 常量按契约；played_at 未来→`ErrInvalidParam`、self→`ErrSelfMatch`、compare>10→`ErrInvalidParam`；main 全程用 `context.Background()` 而非 nil，首启 generated 时同时打印 stdout 并写 `logs/admin_password.txt`。

http 层依赖的 `PlayerService` 会话便利方法 `GetSession(ctx,token)(int64,bool,error)`、`CreatePlayerSession(ctx,playerID)`、`DeletePlayerSession(ctx,token)`、`GetByID(ctx,id)`，以及 `AdminService` 的 `ListDeleted(ctx)([]domain.Match,error)`、`DeleteAdminSession(ctx,token)` **均已在阶段 3 显式实现**（见阶段 3 PlayerService 与 AdminService 的对应方法），方法名与本阶段接口逐字一致，无需适配器。CSRF Origin 校验（Task 14）与管理员登录指数退避（Task 15/16）也已纳入本阶段。

---

## 阶段 5：前端基础（脚手架、依赖、配置、API 层、段位库、路由骨架）

> 本阶段从零搭建 `web/` 前端工程：Vite + React + TypeScript 脚手架、安装全部契约规定的依赖、配置 Tailwind 3 + PostCSS + shadcn/ui、Vite proxy 与 tsconfig 路径别名、axios 客户端与 TS 类型、段位映射纯函数库（严格 TDD）、共享 CSV fixture、路由骨架与路由守卫。
>
> 本阶段所有路径以仓库根 `go_ultra/` 为基准。所有命令在 `web/` 目录下执行（除非另有说明）。Windows 下命令行用 PowerShell，但命令本身跨平台一致。
>
> 强约束（来自契约）：前端 TS 类型字段一律 **snake_case**，与后端 `emit_json_tags` 产出的 JSON 直接对齐，不做大小写转换。`lib/rank.ts` 与 `lib/elo-preview.ts`（阶段 6）是纯函数，**必须严格 TDD**。ECharts 5 色板 hex 固定为 `["#4a9eff", "#7fd6a3", "#8b5cf6", "#e0c47d", "#f08080"]`。段位徽章配色：段 0 灰 / 段 1-3 蓝 / 段 4-6 紫 / 段 7-8 金 / 段 9 红。

### Task 1: 用 Vite 初始化 web 工程并锁定依赖版本

**Files:**
- Create: `web/package.json`（由脚手架生成后修改）
- Create: `web/index.html`、`web/tsconfig.json`、`web/tsconfig.node.json`、`web/vite.config.ts`、`web/src/main.tsx`、`web/src/App.tsx`、`web/src/vite-env.d.ts`（脚手架生成）

步骤：

- [ ] 在仓库根 `go_ultra/` 下执行脚手架命令，在 `web/` 目录生成 react-ts 模板：
  ```bash
  pnpm create vite@latest web --template react-ts
  ```
  预期输出末尾出现 `Scaffolding project in .../web...` 与 `Done. Now run:`。
- [ ] 进入前端目录并安装基础依赖：
  ```bash
  cd web
  pnpm install
  ```
  预期生成 `web/node_modules/` 与 `web/pnpm-lock.yaml`，输出 `Done in ...s`。
- [ ] 安装契约规定的全部运行时依赖（一条命令，确切包名与版本范围）：
  ```bash
  pnpm add react-router-dom@^6.26.0 axios@^1.7.0 @tanstack/react-query@^5.51.0 echarts@^5.5.0 echarts-for-react@^3.0.2 react-hook-form@^7.52.0 zod@^3.23.0 @hookform/resolvers@^3.9.0 sonner@^1.5.0 class-variance-authority@^0.7.0 clsx@^2.1.0 tailwind-merge@^2.4.0 lucide-react@^0.417.0 tailwindcss-animate@^1.0.7
  ```
  预期输出列出 `dependencies:` 并以 `Done in ...s` 结束。
- [ ] 安装契约规定的全部开发依赖（Tailwind 3、PostCSS、Vitest、Testing Library、类型）：
  ```bash
  pnpm add -D tailwindcss@^3.4.0 postcss@^8.4.0 autoprefixer@^10.4.0 vitest@^2.0.0 @testing-library/react@^16.0.0 @testing-library/jest-dom@^6.4.0 @testing-library/user-event@^14.5.0 jsdom@^24.1.0 @types/node@^20.14.0 @vitest/coverage-v8@^2.0.0
  ```
  预期输出列出 `devDependencies:` 并以 `Done in ...s` 结束。
- [ ] 编辑 `web/package.json` 的 `scripts` 段，替换为以下完整内容（保留脚手架生成的 `name`/`private`/`version`/`type`，仅改 `scripts`）：
  ```json
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage"
  }
  ```
- [ ] 验证脚手架可运行（确认 TypeScript 编译无误）：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0（脚手架默认 `App.tsx` 编译通过）。
- [ ] commit：
  ```bash
  git add web/package.json web/pnpm-lock.yaml web/index.html web/tsconfig.json web/tsconfig.node.json web/vite.config.ts web/src
  git commit -m "chore(web): scaffold vite react-ts project and install dependencies"
  ```

### Task 2: 配置 tsconfig 路径别名 @/* 与 vite proxy

**Files:**
- Modify: `web/tsconfig.json`
- Modify: `web/tsconfig.node.json`
- Modify: `web/vite.config.ts`

步骤：

- [ ] 用以下完整内容覆盖 `web/tsconfig.json`（增加 `baseUrl` 与 `paths` 别名 `@/*`，并把 vitest 全局类型纳入）：
  ```json
  {
    "compilerOptions": {
      "target": "ES2020",
      "useDefineForClassFields": true,
      "lib": ["ES2020", "DOM", "DOM.Iterable"],
      "module": "ESNext",
      "skipLibCheck": true,
      "moduleResolution": "bundler",
      "allowImportingTsExtensions": true,
      "resolveJsonModule": true,
      "isolatedModules": true,
      "moduleDetection": "force",
      "noEmit": true,
      "jsx": "react-jsx",
      "strict": true,
      "noUnusedLocals": true,
      "noUnusedParameters": true,
      "noFallthroughCasesInSwitch": true,
      "baseUrl": ".",
      "paths": {
        "@/*": ["./src/*"]
      },
      "types": ["vitest/globals", "@testing-library/jest-dom"]
    },
    "include": ["src"],
    "references": [{ "path": "./tsconfig.node.json" }]
  }
  ```
- [ ] 用以下完整内容覆盖 `web/tsconfig.node.json`：
  ```json
  {
    "compilerOptions": {
      "composite": true,
      "skipLibCheck": true,
      "module": "ESNext",
      "moduleResolution": "bundler",
      "allowSyntheticDefaultImports": true,
      "strict": true,
      "types": ["node"]
    },
    "include": ["vite.config.ts"]
  }
  ```
- [ ] 用以下完整内容覆盖 `web/vite.config.ts`（路径别名 `@` → `src`、proxy `/api` → `http://localhost:8080`、vitest 配置 jsdom + setup + globals）：
  ```ts
  /// <reference types="vitest/config" />
  import { defineConfig } from "vite";
  import react from "@vitejs/plugin-react";
  import path from "node:path";

  export default defineConfig({
    plugins: [react()],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    server: {
      port: 5173,
      proxy: {
        "/api": {
          target: "http://localhost:8080",
          changeOrigin: true,
        },
      },
    },
    test: {
      globals: true,
      environment: "jsdom",
      setupFiles: ["./src/test/setup.ts"],
      css: false,
    },
  });
  ```
- [ ] 创建 vitest setup 文件 `web/src/test/setup.ts`，完整内容：
  ```ts
  import "@testing-library/jest-dom/vitest";
  import { cleanup } from "@testing-library/react";
  import { afterEach } from "vitest";

  afterEach(() => {
    cleanup();
  });
  ```
- [ ] 验证类型检查通过（路径别名与 vitest 类型可解析）：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/tsconfig.json web/tsconfig.node.json web/vite.config.ts web/src/test/setup.ts
  git commit -m "chore(web): configure path alias @/* and vite api proxy"
  ```

### Task 3: 配置 Tailwind 3 + PostCSS + 全局样式

**Files:**
- Create: `web/tailwind.config.ts`
- Create: `web/postcss.config.js`
- Create: `web/src/index.css`（覆盖脚手架默认）
- Modify: `web/src/main.tsx`

步骤：

- [ ] 用以下完整内容创建 `web/postcss.config.js`：
  ```js
  export default {
    plugins: {
      tailwindcss: {},
      autoprefixer: {},
    },
  };
  ```
- [ ] 用以下完整内容创建 `web/tailwind.config.ts`（shadcn/ui 深色主题约定的 CSS 变量配色 + 圆角 + 动画插件 + 5 色 ECharts 调色板作为自定义色，供组件引用）：
  ```ts
  import type { Config } from "tailwindcss";

  const config: Config = {
    darkMode: ["class"],
    content: ["./index.html", "./src/**/*.{ts,tsx}"],
    theme: {
      container: {
        center: true,
        padding: "2rem",
        screens: {
          "2xl": "1400px",
        },
      },
      extend: {
        colors: {
          border: "hsl(var(--border))",
          input: "hsl(var(--input))",
          ring: "hsl(var(--ring))",
          background: "hsl(var(--background))",
          foreground: "hsl(var(--foreground))",
          primary: {
            DEFAULT: "hsl(var(--primary))",
            foreground: "hsl(var(--primary-foreground))",
          },
          secondary: {
            DEFAULT: "hsl(var(--secondary))",
            foreground: "hsl(var(--secondary-foreground))",
          },
          destructive: {
            DEFAULT: "hsl(var(--destructive))",
            foreground: "hsl(var(--destructive-foreground))",
          },
          muted: {
            DEFAULT: "hsl(var(--muted))",
            foreground: "hsl(var(--muted-foreground))",
          },
          accent: {
            DEFAULT: "hsl(var(--accent))",
            foreground: "hsl(var(--accent-foreground))",
          },
          popover: {
            DEFAULT: "hsl(var(--popover))",
            foreground: "hsl(var(--popover-foreground))",
          },
          card: {
            DEFAULT: "hsl(var(--card))",
            foreground: "hsl(var(--card-foreground))",
          },
        },
        borderRadius: {
          lg: "var(--radius)",
          md: "calc(var(--radius) - 2px)",
          sm: "calc(var(--radius) - 4px)",
        },
        keyframes: {
          "accordion-down": {
            from: { height: "0" },
            to: { height: "var(--radix-accordion-content-height)" },
          },
          "accordion-up": {
            from: { height: "var(--radix-accordion-content-height)" },
            to: { height: "0" },
          },
        },
        animation: {
          "accordion-down": "accordion-down 0.2s ease-out",
          "accordion-up": "accordion-up 0.2s ease-out",
        },
      },
    },
    plugins: [require("tailwindcss-animate")],
  };

  export default config;
  ```

  > 注意：上面有两处 `export default`，是笔误。请改用下方“修正版”完整内容，删除冒头的 `const config` 重复导出。

  修正版（请用这一份覆盖 `web/tailwind.config.ts`）：
  ```ts
  import type { Config } from "tailwindcss";
  import animate from "tailwindcss-animate";

  export default {
    darkMode: ["class"],
    content: ["./index.html", "./src/**/*.{ts,tsx}"],
    theme: {
      container: {
        center: true,
        padding: "2rem",
        screens: { "2xl": "1400px" },
      },
      extend: {
        colors: {
          border: "hsl(var(--border))",
          input: "hsl(var(--input))",
          ring: "hsl(var(--ring))",
          background: "hsl(var(--background))",
          foreground: "hsl(var(--foreground))",
          primary: {
            DEFAULT: "hsl(var(--primary))",
            foreground: "hsl(var(--primary-foreground))",
          },
          secondary: {
            DEFAULT: "hsl(var(--secondary))",
            foreground: "hsl(var(--secondary-foreground))",
          },
          destructive: {
            DEFAULT: "hsl(var(--destructive))",
            foreground: "hsl(var(--destructive-foreground))",
          },
          muted: {
            DEFAULT: "hsl(var(--muted))",
            foreground: "hsl(var(--muted-foreground))",
          },
          accent: {
            DEFAULT: "hsl(var(--accent))",
            foreground: "hsl(var(--accent-foreground))",
          },
          popover: {
            DEFAULT: "hsl(var(--popover))",
            foreground: "hsl(var(--popover-foreground))",
          },
          card: {
            DEFAULT: "hsl(var(--card))",
            foreground: "hsl(var(--card-foreground))",
          },
        },
        borderRadius: {
          lg: "var(--radius)",
          md: "calc(var(--radius) - 2px)",
          sm: "calc(var(--radius) - 4px)",
        },
        keyframes: {
          "accordion-down": {
            from: { height: "0" },
            to: { height: "var(--radix-accordion-content-height)" },
          },
          "accordion-up": {
            from: { height: "var(--radix-accordion-content-height)" },
            to: { height: "0" },
          },
        },
        animation: {
          "accordion-down": "accordion-down 0.2s ease-out",
          "accordion-up": "accordion-up 0.2s ease-out",
        },
      },
    },
    plugins: [animate],
  } satisfies Config;
  ```
- [ ] 用以下完整内容覆盖 `web/src/index.css`（Tailwind 三层指令 + shadcn/ui 深色主题 CSS 变量，默认强制深色）：
  ```css
  @tailwind base;
  @tailwind components;
  @tailwind utilities;

  @layer base {
    :root {
      --background: 240 10% 4%;
      --foreground: 0 0% 98%;
      --card: 240 10% 6%;
      --card-foreground: 0 0% 98%;
      --popover: 240 10% 6%;
      --popover-foreground: 0 0% 98%;
      --primary: 0 0% 98%;
      --primary-foreground: 240 6% 10%;
      --secondary: 240 4% 16%;
      --secondary-foreground: 0 0% 98%;
      --muted: 240 4% 16%;
      --muted-foreground: 240 5% 65%;
      --accent: 240 4% 16%;
      --accent-foreground: 0 0% 98%;
      --destructive: 0 63% 31%;
      --destructive-foreground: 0 0% 98%;
      --border: 240 4% 16%;
      --input: 240 4% 16%;
      --ring: 240 5% 84%;
      --radius: 0.5rem;
    }
  }

  @layer base {
    * {
      @apply border-border;
    }
    html {
      @apply dark;
    }
    body {
      @apply bg-background text-foreground;
      font-feature-settings: "rlig" 1, "calt" 1;
    }
  }
  ```
- [ ] 用以下完整内容覆盖 `web/src/main.tsx`（引入全局样式、挂载 React Query Provider 与 Sonner toaster、BrowserRouter）：
  ```tsx
  import React from "react";
  import ReactDOM from "react-dom/client";
  import { BrowserRouter } from "react-router-dom";
  import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
  import { Toaster } from "sonner";
  import App from "@/App";
  import "@/index.css";

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: 1,
        refetchOnWindowFocus: false,
      },
    },
  });

  ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <App />
        </BrowserRouter>
        <Toaster richColors position="top-right" />
      </QueryClientProvider>
    </React.StrictMode>,
  );
  ```
- [ ] 删除脚手架自带的 `web/src/App.css`（不再使用，避免与 Tailwind 冲突）：
  ```bash
  rm web/src/App.css
  ```
- [ ] 暂时把脚手架的 `web/src/App.tsx` 替换为最小占位（Task 13 会写真正路由），完整内容：
  ```tsx
  export default function App() {
    return <div className="p-8 text-2xl font-bold">go_ultra</div>;
  }
  ```
- [ ] 启动开发服务器手动验证 Tailwind 生效（仅本步骤，确认后 Ctrl+C 退出）：
  ```bash
  pnpm dev
  ```
  预期输出包含 `VITE v5.x  ready in ...ms` 与 `Local: http://localhost:5173/`，浏览器打开后能看到深色背景上的 “go_ultra” 大字。
- [ ] commit：
  ```bash
  git add web/tailwind.config.ts web/postcss.config.js web/src/index.css web/src/main.tsx web/src/App.tsx
  git rm --cached web/src/App.css
  git commit -m "feat(web): set up tailwind dark theme, react-query and toaster"
  ```

### Task 4: 初始化 shadcn/ui（components.json + cn 工具 + 基础组件）

**Files:**
- Create: `web/components.json`
- Create: `web/src/lib/utils.ts`
- Create: `web/src/components/ui/button.tsx`、`web/src/components/ui/card.tsx`、`web/src/components/ui/input.tsx`、`web/src/components/ui/dialog.tsx`、`web/src/components/ui/table.tsx`、`web/src/components/ui/select.tsx`、`web/src/components/ui/label.tsx`、`web/src/components/ui/command.tsx`、`web/src/components/ui/popover.tsx`、`web/src/components/ui/badge.tsx`、`web/src/components/ui/tooltip.tsx`、`web/src/components/ui/dropdown-menu.tsx`

> 说明：shadcn/ui 不是 npm 依赖而是“复制进项目”的组件源码。本任务先写 `components.json` 与 `cn` 工具，再用 `pnpm dlx shadcn@latest add` 拉取所需基础组件。该命令会自动安装对应的 Radix 依赖（`@radix-ui/react-*`）。

步骤：

- [ ] 用以下完整内容创建 `web/components.json`（new-york 风格、深色、别名指向 `@`）：
  ```json
  {
    "$schema": "https://ui.shadcn.com/schema.json",
    "style": "new-york",
    "rsc": false,
    "tsx": true,
    "tailwind": {
      "config": "tailwind.config.ts",
      "css": "src/index.css",
      "baseColor": "zinc",
      "cssVariables": true,
      "prefix": ""
    },
    "aliases": {
      "components": "@/components",
      "utils": "@/lib/utils",
      "ui": "@/components/ui",
      "lib": "@/lib",
      "hooks": "@/hooks"
    }
  }
  ```
- [ ] 用以下完整内容创建 `web/src/lib/utils.ts`（shadcn/ui 标准 `cn` 合并工具）：
  ```ts
  import { clsx, type ClassValue } from "clsx";
  import { twMerge } from "tailwind-merge";

  export function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
  }
  ```
- [ ] 拉取本项目用到的全部 shadcn 基础组件（一条命令，自动安装 Radix 依赖）：
  ```bash
  pnpm dlx shadcn@latest add button card input dialog table select label command popover badge tooltip dropdown-menu --yes
  ```
  预期输出逐个列出 `✔ Created ... src/components/ui/<name>.tsx` 并在 `package.json` 增加 `@radix-ui/react-dialog`、`@radix-ui/react-select`、`@radix-ui/react-popover`、`@radix-ui/react-tooltip`、`@radix-ui/react-dropdown-menu`、`@radix-ui/react-label`、`cmdk` 等依赖。
- [ ] 验证生成的组件可被 TypeScript 解析：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/components.json web/src/lib/utils.ts web/src/components/ui web/package.json web/pnpm-lock.yaml
  git commit -m "feat(web): init shadcn/ui and add base ui components"
  ```

### Task 5: 共享段位 fixture（rank_cases.csv）

**Files:**
- Create: `web/src/lib/__fixtures__/rank_cases.csv`
- Create: `server/internal/domain/testdata/rank_cases.csv`

> 该 CSV 是前后端段位映射的**同一份真相源**（契约要求两处内容一致）。两列 `rating,expected_dan`，第一行为表头。覆盖契约第 83 行列出的全部边界。

步骤：

- [ ] 用以下完整内容创建 `web/src/lib/__fixtures__/rank_cases.csv`：
  ```csv
  rating,expected_dan
  1049,0
  1050,1
  1199,1
  1200,2
  1399,2
  1400,3
  1500,3
  1599,3
  1600,4
  2399,7
  2400,8
  2599,8
  2600,9
  5000,9
  ```
- [ ] 用**完全相同**的内容创建 `server/internal/domain/testdata/rank_cases.csv`（后端 domain 测试将读取同一份数据，保证前后端一致）：
  ```csv
  rating,expected_dan
  1049,0
  1050,1
  1199,1
  1200,2
  1399,2
  1400,3
  1500,3
  1599,3
  1600,4
  2399,7
  2400,8
  2599,8
  2600,9
  5000,9
  ```
- [ ] 验证两文件字节一致（防止跑偏；预期无输出表示一致）：
  ```bash
  diff web/src/lib/__fixtures__/rank_cases.csv server/internal/domain/testdata/rank_cases.csv
  ```
  预期：无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/lib/__fixtures__/rank_cases.csv server/internal/domain/testdata/rank_cases.csv
  git commit -m "test(web): add shared rank mapping fixture csv"
  ```

### Task 6: lib/rank.ts —— 段位映射纯函数（严格 TDD）

**Files:**
- Create: `web/src/lib/rank.ts`
- Test: `web/src/lib/rank.test.ts`

> 严格 TDD：先写读 CSV fixture 的失败测试，确认失败，再写最小实现，确认通过。函数签名严格按契约：`danOf(rating)`、`danLabel(dan)`、`danColor(dan)`。
> `danColor` 配色按 spec §7.5：段 0 灰 / 1-3 蓝 / 4-6 紫 / 7-8 金 / 9 红。具体 hex 固定如下（实现与测试用同一组值）：
> - 段 0：`#9ca3af`（灰）
> - 段 1-3：`#4a9eff`（蓝）
> - 段 4-6：`#8b5cf6`（紫）
> - 段 7-8：`#e0c47d`（金）
> - 段 9：`#f08080`（红）

步骤：

- [ ] 先写失败测试 `web/src/lib/rank.test.ts`，完整内容（读共享 CSV fixture 驱动 `danOf`，并覆盖 `danLabel`/`danColor`）：
  ```ts
  import { describe, it, expect } from "vitest";
  import fs from "node:fs";
  import path from "node:path";
  import { danOf, danLabel, danColor } from "@/lib/rank";

  function loadCases(): { rating: number; expected: number }[] {
    const csvPath = path.resolve(
      __dirname,
      "./__fixtures__/rank_cases.csv",
    );
    const raw = fs.readFileSync(csvPath, "utf-8").trim();
    const lines = raw.split(/\r?\n/);
    // skip header
    return lines.slice(1).map((line) => {
      const [rating, expected] = line.split(",");
      return { rating: Number(rating), expected: Number(expected) };
    });
  }

  describe("danOf", () => {
    const cases = loadCases();
    it("loads at least the documented boundary cases", () => {
      expect(cases.length).toBeGreaterThanOrEqual(14);
    });
    it.each(cases)(
      "rating $rating -> dan $expected",
      ({ rating, expected }) => {
        expect(danOf(rating)).toBe(expected);
      },
    );
  });

  describe("danLabel", () => {
    it("returns 未定级 for dan 0", () => {
      expect(danLabel(0)).toBe("未定级");
    });
    it.each([
      [1, "段 1"],
      [3, "段 3"],
      [9, "段 9"],
    ])("dan %i -> %s", (dan, label) => {
      expect(danLabel(dan)).toBe(label);
    });
  });

  describe("danColor", () => {
    it.each([
      [0, "#9ca3af"],
      [1, "#4a9eff"],
      [2, "#4a9eff"],
      [3, "#4a9eff"],
      [4, "#8b5cf6"],
      [5, "#8b5cf6"],
      [6, "#8b5cf6"],
      [7, "#e0c47d"],
      [8, "#e0c47d"],
      [9, "#f08080"],
    ])("dan %i -> %s", (dan, hex) => {
      expect(danColor(dan)).toBe(hex);
    });
  });
  ```
- [ ] 运行测试确认失败（实现尚不存在）：
  ```bash
  pnpm test src/lib/rank.test.ts
  ```
  预期失败，错误信息形如 `Failed to resolve import "@/lib/rank"` 或 `does not provide an export named 'danOf'`。
- [ ] 写最小实现 `web/src/lib/rank.ts`，完整内容（`danOf` 与后端 `Dan` 完全一致）：
  ```ts
  export const RANK_FLOOR = 1050;

  /**
   * 与后端 domain.Dan 完全一致的段位映射。
   * rating < 1050 -> 0；否则 (rating-800)/200 向下取整，超过 9 截断为 9。
   */
  export function danOf(rating: number): number {
    if (rating < RANK_FLOOR) {
      return 0;
    }
    const tier = Math.floor((rating - 800) / 200);
    if (tier > 9) {
      return 9;
    }
    return tier;
  }

  export function danLabel(dan: number): string {
    return dan === 0 ? "未定级" : `段 ${dan}`;
  }

  /** 段位徽章配色：段0灰 / 1-3蓝 / 4-6紫 / 7-8金 / 9红 */
  export function danColor(dan: number): string {
    if (dan === 0) return "#9ca3af";
    if (dan <= 3) return "#4a9eff";
    if (dan <= 6) return "#8b5cf6";
    if (dan <= 8) return "#e0c47d";
    return "#f08080";
  }
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/lib/rank.test.ts
  ```
  预期输出 `Test Files  1 passed`、所有 case 绿色通过。
- [ ] commit：
  ```bash
  git add web/src/lib/rank.ts web/src/lib/rank.test.ts
  git commit -m "feat(web): add rank mapping lib with shared-fixture tests"
  ```

### Task 7: api/types.ts —— 与后端 JSON 对齐的 TS 类型（snake_case）

**Files:**
- Create: `web/src/api/types.ts`

> 严格按 spec §6 的响应体与契约 service 视图字段命名，全部 snake_case，与后端 `emit_json_tags` 的 JSON 直接对齐，不做转换。

步骤：

- [ ] 用以下完整内容创建 `web/src/api/types.ts`：
  ```ts
  // 所有字段命名与后端 JSON（snake_case）一一对应，前端不做转换。

  export interface Player {
    id: number;
    username: string;
    rating: number;
    dan: number;
    created_at: string;
  }

  export interface PlayerStats {
    wins: number;
    losses: number;
    win_rate: number;
    current_streak: number;
    longest_streak: number;
  }

  export interface PlayerDetail {
    id: number;
    username: string;
    rating: number;
    dan: number;
    created_at: string;
    stats: PlayerStats;
  }

  export interface LeaderboardRow {
    rank: number;
    username: string;
    rating: number;
    dan: number;
    games_played: number;
    win_rate: number;
  }

  export interface HistoryPoint {
    played_at: string;
    rating: number;
  }

  export type MatchResult = "win" | "loss";

  export interface MatchView {
    id: number;
    opponent: string;
    result: MatchResult;
    rating_before: number;
    rating_after: number;
    delta: number;
    played_at: string;
  }

  export interface RecordMatchRequest {
    opponent_username: string;
    result: MatchResult;
    played_at?: string;
  }

  export interface RecordMatchResponse {
    id: number;
    winner_delta: number;
    loser_delta: number;
    new_self_rating: number;
    new_opponent_rating: number;
  }

  export interface CompareSeries {
    username: string;
    color: string;
    points: HistoryPoint[];
  }

  export interface HeadToHead {
    a: string;
    b: string;
    a_wins: number;
    b_wins: number;
  }

  export interface CompareResult {
    series: CompareSeries[];
    head_to_head: HeadToHead[];
  }

  export interface AdminStatus {
    authed: boolean;
    expires_at?: string;
  }

  export interface AdminLoginResponse {
    expires_at: string;
  }

  export interface ApiErrorBody {
    error: {
      code: string;
      message: string;
    };
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/api/types.ts
  git commit -m "feat(web): add api dto types aligned with backend json"
  ```

### Task 8: api/client.ts —— axios 实例 + 错误 interceptor（含 ApiError 单测）

**Files:**
- Create: `web/src/api/client.ts`
- Test: `web/src/api/client.test.ts`

> 契约要求：baseURL `/api`，`withCredentials: true`；响应错误 interceptor 解析 `{error:{code,message}}` → 抛 `ApiError`；401 → 跳 `/login`。
> 这里把“解析后端错误体”的纯逻辑抽成可单测的 `parseApiError(error)`，interceptor 复用它；401 跳转用 `window.location.assign`（便于断言）。

步骤：

- [ ] 先写失败测试 `web/src/api/client.test.ts`，完整内容（验证 `ApiError` 与 `parseApiError` 解析三种情况：标准错误体、无错误体、网络无响应）：
  ```ts
  import { describe, it, expect } from "vitest";
  import { ApiError, parseApiError } from "@/api/client";

  describe("parseApiError", () => {
    it("parses backend { error: { code, message } } body", () => {
      const err = parseApiError({
        response: {
          status: 404,
          data: { error: { code: "PLAYER_NOT_FOUND", message: "玩家不存在" } },
        },
      });
      expect(err).toBeInstanceOf(ApiError);
      expect(err.code).toBe("PLAYER_NOT_FOUND");
      expect(err.message).toBe("玩家不存在");
      expect(err.status).toBe(404);
    });

    it("falls back to UNKNOWN when body has no error envelope", () => {
      const err = parseApiError({
        response: { status: 500, data: { something: "else" } },
      });
      expect(err.code).toBe("UNKNOWN");
      expect(err.status).toBe(500);
    });

    it("handles network error with no response", () => {
      const err = parseApiError({ message: "Network Error" });
      expect(err.code).toBe("NETWORK_ERROR");
      expect(err.status).toBe(0);
    });
  });
  ```
- [ ] 运行测试确认失败：
  ```bash
  pnpm test src/api/client.test.ts
  ```
  预期失败，错误信息形如 `Failed to resolve import "@/api/client"`。
- [ ] 写最小实现 `web/src/api/client.ts`，完整内容：
  ```ts
  import axios, { type AxiosError } from "axios";
  import type { ApiErrorBody } from "@/api/types";

  export class ApiError extends Error {
    code: string;
    status: number;
    constructor(code: string, message: string, status: number) {
      super(message);
      this.name = "ApiError";
      this.code = code;
      this.status = status;
    }
  }

  /** 把 axios 抛出的错误解析为统一的 ApiError。纯函数，便于单测。 */
  export function parseApiError(error: unknown): ApiError {
    const ax = error as Partial<AxiosError<ApiErrorBody>>;
    const response = ax.response;
    if (response) {
      const body = response.data as ApiErrorBody | undefined;
      if (body && body.error && body.error.code) {
        return new ApiError(
          body.error.code,
          body.error.message,
          response.status,
        );
      }
      return new ApiError("UNKNOWN", "未知错误", response.status);
    }
    return new ApiError("NETWORK_ERROR", "网络错误", 0);
  }

  export const client = axios.create({
    baseURL: "/api",
    withCredentials: true,
    headers: { "Content-Type": "application/json" },
  });

  client.interceptors.response.use(
    (res) => res,
    (error) => {
      const apiError = parseApiError(error);
      if (apiError.status === 401) {
        if (window.location.pathname !== "/login") {
          window.location.assign("/login");
        }
      }
      return Promise.reject(apiError);
    },
  );
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/api/client.test.ts
  ```
  预期 `Test Files  1 passed`，3 个 case 通过。
- [ ] commit：
  ```bash
  git add web/src/api/client.ts web/src/api/client.test.ts
  git commit -m "feat(web): add axios client with error interceptor and ApiError"
  ```

### Task 9: api/players.ts —— 玩家相关请求封装

**Files:**
- Create: `web/src/api/players.ts`

> 端点严格按 spec §6.1 玩家段。函数返回后端 DTO 类型（来自 `types.ts`）。`/api/me`、`/api/login`、`/api/logout` 也归到此文件（玩家身份相关）。

步骤：

- [ ] 用以下完整内容创建 `web/src/api/players.ts`：
  ```ts
  import { client } from "@/api/client";
  import type {
    Player,
    PlayerDetail,
    HistoryPoint,
    MatchView,
  } from "@/api/types";

  export async function login(username: string): Promise<Player> {
    const res = await client.post<{ player: Player }>("/login", { username });
    return res.data.player;
  }

  export async function logout(): Promise<void> {
    await client.post("/logout");
  }

  export async function getMe(): Promise<Player> {
    const res = await client.get<{ player: Player }>("/me");
    return res.data.player;
  }

  export async function listPlayers(): Promise<Player[]> {
    const res = await client.get<Player[]>("/players");
    return res.data;
  }

  export async function getPlayer(username: string): Promise<PlayerDetail> {
    const res = await client.get<PlayerDetail>(
      `/players/${encodeURIComponent(username)}`,
    );
    return res.data;
  }

  export async function getPlayerHistory(
    username: string,
    params?: { from?: string; to?: string },
  ): Promise<HistoryPoint[]> {
    const res = await client.get<HistoryPoint[]>(
      `/players/${encodeURIComponent(username)}/history`,
      { params },
    );
    return res.data;
  }

  export async function getPlayerMatches(
    username: string,
    params?: { limit?: number; offset?: number },
  ): Promise<MatchView[]> {
    const res = await client.get<MatchView[]>(
      `/players/${encodeURIComponent(username)}/matches`,
      { params },
    );
    return res.data;
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/api/players.ts
  git commit -m "feat(web): add players api module"
  ```

### Task 10: api/matches.ts —— 对局与排行榜/对比请求封装

**Files:**
- Create: `web/src/api/matches.ts`

> 端点严格按 spec §6.1 对局/排行榜/对比段。`compare` 用 `usernames` 逗号拼接（契约后端上限 10）。

步骤：

- [ ] 用以下完整内容创建 `web/src/api/matches.ts`：
  ```ts
  import { client } from "@/api/client";
  import type {
    MatchView,
    RecordMatchRequest,
    RecordMatchResponse,
    LeaderboardRow,
    CompareResult,
  } from "@/api/types";

  export async function recordMatch(
    body: RecordMatchRequest,
  ): Promise<RecordMatchResponse> {
    const res = await client.post<RecordMatchResponse>("/matches", body);
    return res.data;
  }

  export async function listGlobalMatches(params?: {
    limit?: number;
    offset?: number;
  }): Promise<MatchView[]> {
    const res = await client.get<MatchView[]>("/matches", { params });
    return res.data;
  }

  export async function getLeaderboard(
    minGames = 0,
  ): Promise<LeaderboardRow[]> {
    const res = await client.get<LeaderboardRow[]>("/leaderboard", {
      params: { min_games: minGames },
    });
    return res.data;
  }

  export async function getCompare(
    usernames: string[],
    params?: { from?: string; to?: string },
  ): Promise<CompareResult> {
    const res = await client.get<CompareResult>("/compare", {
      params: { usernames: usernames.join(","), ...params },
    });
    return res.data;
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/api/matches.ts
  git commit -m "feat(web): add matches/leaderboard/compare api module"
  ```

### Task 11: api/admin.ts —— 管理员请求封装

**Files:**
- Create: `web/src/api/admin.ts`

> 端点严格按 spec §6.1 鉴权 + 对局管理段。已删除列表复用 `MatchView`（后端返回同结构）。

步骤：

- [ ] 用以下完整内容创建 `web/src/api/admin.ts`：
  ```ts
  import { client } from "@/api/client";
  import type {
    AdminStatus,
    AdminLoginResponse,
    MatchView,
  } from "@/api/types";

  export async function adminLogin(
    password: string,
  ): Promise<AdminLoginResponse> {
    const res = await client.post<AdminLoginResponse>("/admin/login", {
      password,
    });
    return res.data;
  }

  export async function adminLogout(): Promise<void> {
    await client.post("/admin/logout");
  }

  export async function getAdminStatus(): Promise<AdminStatus> {
    const res = await client.get<AdminStatus>("/admin/status");
    return res.data;
  }

  export async function deleteMatch(id: number): Promise<void> {
    await client.delete(`/matches/${id}`);
  }

  export async function listDeletedMatches(): Promise<MatchView[]> {
    const res = await client.get<MatchView[]>("/admin/matches/deleted");
    return res.data;
  }

  export async function restoreMatch(id: number): Promise<void> {
    await client.post(`/admin/matches/${id}/restore`);
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/api/admin.ts
  git commit -m "feat(web): add admin api module"
  ```

### Task 12: hooks/useAuth.ts —— 当前用户会话 hook

**Files:**
- Create: `web/src/hooks/useAuth.ts`

> 用 React Query 查询 `/api/me`，401 时 `getMe` 会被 client 拦截器跳转，这里把 401 视为“未登录”而不重试。供 `AuthGuard` 与 `Layout` 使用。

步骤：

- [ ] 用以下完整内容创建 `web/src/hooks/useAuth.ts`：
  ```ts
  import { useQuery } from "@tanstack/react-query";
  import { getMe } from "@/api/players";
  import { ApiError } from "@/api/client";
  import type { Player } from "@/api/types";

  export function useAuth() {
    const query = useQuery<Player, ApiError>({
      queryKey: ["me"],
      queryFn: getMe,
      retry: false,
      staleTime: 60_000,
    });

    return {
      player: query.data ?? null,
      isLoading: query.isLoading,
      isAuthenticated: !!query.data,
      error: query.error,
    };
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/hooks/useAuth.ts
  git commit -m "feat(web): add useAuth hook backed by /api/me"
  ```

### Task 13: 路由守卫 AuthGuard / AdminGuard + App.tsx 路由骨架

**Files:**
- Create: `web/src/components/AuthGuard.tsx`
- Create: `web/src/components/AdminGuard.tsx`
- Create: `web/src/pages/Login.tsx`、`web/src/pages/Dashboard.tsx`、`web/src/pages/Leaderboard.tsx`、`web/src/pages/PlayerDetail.tsx`、`web/src/pages/Compare.tsx`、`web/src/pages/Admin.tsx`（均为占位骨架，阶段 6 填充）
- Modify: `web/src/App.tsx`
- Test: `web/src/components/AuthGuard.test.tsx`

> 本任务建立完整路由骨架与守卫。页面先用占位（阶段 6 实现真正内容）。`AuthGuard` 写组件测试（未登录重定向、加载中显示占位、已登录渲染子节点）。`AdminGuard` 查询 `/api/admin/status`，未授权渲染密码框（占位，阶段 6 在 Admin 页完善交互）。

步骤：

- [ ] 创建六个页面占位文件。`web/src/pages/Login.tsx`：
  ```tsx
  export default function Login() {
    return <div data-testid="page-login">Login</div>;
  }
  ```
  `web/src/pages/Dashboard.tsx`：
  ```tsx
  export default function Dashboard() {
    return <div data-testid="page-dashboard">Dashboard</div>;
  }
  ```
  `web/src/pages/Leaderboard.tsx`：
  ```tsx
  export default function Leaderboard() {
    return <div data-testid="page-leaderboard">Leaderboard</div>;
  }
  ```
  `web/src/pages/PlayerDetail.tsx`：
  ```tsx
  export default function PlayerDetail() {
    return <div data-testid="page-player-detail">PlayerDetail</div>;
  }
  ```
  `web/src/pages/Compare.tsx`：
  ```tsx
  export default function Compare() {
    return <div data-testid="page-compare">Compare</div>;
  }
  ```
  `web/src/pages/Admin.tsx`：
  ```tsx
  export default function Admin() {
    return <div data-testid="page-admin">Admin</div>;
  }
  ```
- [ ] 先写 `AuthGuard` 失败测试 `web/src/components/AuthGuard.test.tsx`，完整内容：
  ```tsx
  import { describe, it, expect, vi, beforeEach } from "vitest";
  import { render, screen } from "@testing-library/react";
  import { MemoryRouter, Routes, Route } from "react-router-dom";
  import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
  import AuthGuard from "@/components/AuthGuard";
  import * as playersApi from "@/api/players";

  function renderWithAuth(initialPath: string) {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    return render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={[initialPath]}>
          <Routes>
            <Route
              path="/me"
              element={
                <AuthGuard>
                  <div data-testid="protected">secret</div>
                </AuthGuard>
              }
            />
            <Route path="/login" element={<div data-testid="login">login</div>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
  }

  describe("AuthGuard", () => {
    beforeEach(() => {
      vi.restoreAllMocks();
    });

    it("renders children when authenticated", async () => {
      vi.spyOn(playersApi, "getMe").mockResolvedValue({
        id: 1,
        username: "alice",
        rating: 1500,
        dan: 3,
        created_at: "2026-06-25T00:00:00Z",
      });
      renderWithAuth("/me");
      expect(await screen.findByTestId("protected")).toBeInTheDocument();
    });

    it("redirects to /login when unauthenticated", async () => {
      vi.spyOn(playersApi, "getMe").mockRejectedValue(new Error("401"));
      renderWithAuth("/me");
      expect(await screen.findByTestId("login")).toBeInTheDocument();
    });
  });
  ```
- [ ] 运行测试确认失败：
  ```bash
  pnpm test src/components/AuthGuard.test.tsx
  ```
  预期失败，错误信息形如 `Failed to resolve import "@/components/AuthGuard"`。
- [ ] 写最小实现 `web/src/components/AuthGuard.tsx`，完整内容：
  ```tsx
  import type { ReactNode } from "react";
  import { Navigate } from "react-router-dom";
  import { useAuth } from "@/hooks/useAuth";

  export default function AuthGuard({ children }: { children: ReactNode }) {
    const { isAuthenticated, isLoading } = useAuth();

    if (isLoading) {
      return (
        <div className="flex h-screen items-center justify-center text-muted-foreground">
          加载中…
        </div>
      );
    }
    if (!isAuthenticated) {
      return <Navigate to="/login" replace />;
    }
    return <>{children}</>;
  }
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/components/AuthGuard.test.tsx
  ```
  预期 `Test Files  1 passed`，2 个 case 通过。
- [ ] 写 `AdminGuard` 实现 `web/src/components/AdminGuard.tsx`，完整内容（未授权时渲染最小密码框占位，阶段 6 的 Admin 页会完善体验；这里只保证守卫与状态查询正确）：
  ```tsx
  import { useState, type ReactNode } from "react";
  import { useQuery } from "@tanstack/react-query";
  import { getAdminStatus, adminLogin } from "@/api/admin";
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import { useQueryClient } from "@tanstack/react-query";
  import { toast } from "sonner";
  import { ApiError } from "@/api/client";

  export default function AdminGuard({ children }: { children: ReactNode }) {
    const qc = useQueryClient();
    const [password, setPassword] = useState("");
    const { data, isLoading } = useQuery({
      queryKey: ["admin-status"],
      queryFn: getAdminStatus,
      retry: false,
    });

    async function handleSubmit(e: React.FormEvent) {
      e.preventDefault();
      try {
        await adminLogin(password);
        await qc.invalidateQueries({ queryKey: ["admin-status"] });
        toast.success("管理员已登录");
      } catch (err) {
        const code = err instanceof ApiError ? err.code : "UNKNOWN";
        toast.error(code === "RATE_LIMITED" ? "尝试过于频繁，请稍后" : "密码错误");
      }
    }

    if (isLoading) {
      return (
        <div className="flex h-screen items-center justify-center text-muted-foreground">
          加载中…
        </div>
      );
    }
    if (!data?.authed) {
      return (
        <div className="flex h-screen items-center justify-center">
          <form
            onSubmit={handleSubmit}
            className="w-80 space-y-4 rounded-lg border bg-card p-6"
          >
            <h2 className="text-lg font-semibold">管理员登录</h2>
            <div className="space-y-2">
              <Label htmlFor="admin-password">密码</Label>
              <Input
                id="admin-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="off"
              />
            </div>
            <Button type="submit" className="w-full">
              登录
            </Button>
          </form>
        </div>
      );
    }
    return <>{children}</>;
  }
  ```
- [ ] 用以下完整内容覆盖 `web/src/App.tsx`（路由表严格按 spec §7.1：`/` 已登录跳 `/me` 否则 `/login`；受保护页用 `AuthGuard`；`/admin` 套 `AdminGuard`）：
  ```tsx
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
  ```
- [ ] 全量类型检查与测试通过：
  ```bash
  pnpm exec tsc -b && pnpm test
  ```
  预期 `tsc` 无输出退出码 0；`pnpm test` 显示 rank / client / AuthGuard 全部通过。
- [ ] commit：
  ```bash
  git add web/src/components/AuthGuard.tsx web/src/components/AuthGuard.test.tsx web/src/components/AdminGuard.tsx web/src/pages web/src/App.tsx
  git commit -m "feat(web): add route guards and routing skeleton"
  ```

---

## 阶段 6：前端页面与组件（Layout、图表、表格、弹窗、各页面）

> 本阶段在阶段 5 的基础上实现全部业务组件与页面。组件与页面布局严格按 spec §7.2–§7.5：我的主页 = 大曲线 + 右侧栏（布局 B）；排行榜 = 表格 + 顶部领奖台（布局 A）；对比 = 左侧栏控制 + 右大曲线 + 下方头对头卡片（B+C 合并）；录入 = 标准表单 + Elo 预览（布局 A）。
>
> ECharts 暗色主题；5 色板 hex 固定为 `["#4a9eff", "#7fd6a3", "#8b5cf6", "#e0c47d", "#f08080"]`。
>
> TDD 约束：`lib/elo-preview.ts` 是纯函数，**必须严格 TDD**（先测后写），且与后端 `ComputeDelta` 同公式。纯展示组件（Layout、RankBadge、RatingChart、CompareChart、MatchTable、PlayerCombobox）可“先写组件再写测试”。`RankBadge` 与 `SubmitMatchDialog` 必须配 vitest 组件测试。

### Task 14: lib/elo-preview.ts —— Elo 预览纯函数（严格 TDD）

**Files:**
- Create: `web/src/lib/elo-preview.ts`
- Test: `web/src/lib/elo-preview.test.ts`

> 严格 TDD。必须与后端 `domain.ComputeDelta` 同公式：`round(K*(1-E_winner))`，`E_winner = 1/(1+10^((loser-winner)/400))`，`K=16`，`round` 为 half-away-from-zero（与 Go `math.Round` 一致）。loser delta = `-delta`（零和）。
> 提供 `expectedScore(a,b)`、`computeDelta(winner,loser)`、`previewMatch(selfRating, opponentRating, result)`（result 为 self 视角 "win"|"loss"，返回 self/opponent 的 before/after/delta）。

步骤：

- [ ] 先写失败测试 `web/src/lib/elo-preview.test.ts`，完整内容（边界对照与后端一致：分差 0 → delta 8；自己视角 win/loss；零和性）：
  ```ts
  import { describe, it, expect } from "vitest";
  import { expectedScore, computeDelta, previewMatch } from "@/lib/elo-preview";

  describe("expectedScore", () => {
    it("equal ratings -> 0.5", () => {
      expect(expectedScore(1500, 1500)).toBeCloseTo(0.5, 10);
    });
    it("400 higher -> ~0.909", () => {
      expect(expectedScore(1900, 1500)).toBeCloseTo(0.9090909, 6);
    });
  });

  describe("computeDelta", () => {
    it.each([
      [1500, 1500, 8],
      [1900, 1500, 1],
      [1500, 1900, 15],
    ])("computeDelta(%i,%i) = %i", (winner, loser, expected) => {
      expect(computeDelta(winner, loser)).toBe(expected);
    });

    it("uses half-away-from-zero rounding (matches Go math.Round)", () => {
      // 构造一个 K*(1-E) 恰为 .5 的情形难以保证，改为验证非负且整数
      const d = computeDelta(1600, 1450);
      expect(Number.isInteger(d)).toBe(true);
      expect(d).toBeGreaterThan(0);
    });
  });

  describe("previewMatch", () => {
    it("self win is zero-sum", () => {
      const r = previewMatch(1500, 1500, "win");
      expect(r.self_delta).toBe(8);
      expect(r.opponent_delta).toBe(-8);
      expect(r.self_after).toBe(1508);
      expect(r.opponent_after).toBe(1492);
      expect(r.self_delta + r.opponent_delta).toBe(0);
    });

    it("self loss: opponent is winner", () => {
      const r = previewMatch(1500, 1500, "loss");
      // opponent wins +8 vs equal rating; self loses 8
      expect(r.self_delta).toBe(-8);
      expect(r.opponent_delta).toBe(8);
      expect(r.self_after).toBe(1492);
      expect(r.opponent_after).toBe(1508);
    });

    it("upset win against stronger opponent gains more", () => {
      const r = previewMatch(1500, 1900, "win");
      expect(r.self_delta).toBe(15);
      expect(r.opponent_delta).toBe(-15);
    });
  });
  ```
- [ ] 运行测试确认失败：
  ```bash
  pnpm test src/lib/elo-preview.test.ts
  ```
  预期失败，错误信息形如 `Failed to resolve import "@/lib/elo-preview"`。
- [ ] 写最小实现 `web/src/lib/elo-preview.ts`，完整内容：
  ```ts
  export const K_FACTOR = 16;
  export const DEFAULT_RATING = 1500;

  /** E_A = 1 / (1 + 10^((B - A) / 400))，与后端 ExpectedScore 一致 */
  export function expectedScore(ratingA: number, ratingB: number): number {
    return 1 / (1 + Math.pow(10, (ratingB - ratingA) / 400));
  }

  /** half-away-from-zero 取整，与 Go math.Round 一致 */
  function roundHalfAway(x: number): number {
    return Math.sign(x) * Math.round(Math.abs(x));
  }

  /** round(K * (1 - E_winner))，与后端 ComputeDelta 一致 */
  export function computeDelta(winnerRating: number, loserRating: number): number {
    const eWinner = expectedScore(winnerRating, loserRating);
    return roundHalfAway(K_FACTOR * (1 - eWinner));
  }

  export type SelfResult = "win" | "loss";

  export interface MatchPreview {
    self_before: number;
    opponent_before: number;
    self_after: number;
    opponent_after: number;
    self_delta: number;
    opponent_delta: number;
  }

  /**
   * 以 self 视角预览一局结果。
   * result=win => self 是赢家；result=loss => opponent 是赢家。
   * 与后端 MatchService.Record 的快照语义一致（零和）。
   */
  export function previewMatch(
    selfRating: number,
    opponentRating: number,
    result: SelfResult,
  ): MatchPreview {
    if (result === "win") {
      const delta = computeDelta(selfRating, opponentRating);
      return {
        self_before: selfRating,
        opponent_before: opponentRating,
        self_after: selfRating + delta,
        opponent_after: opponentRating - delta,
        self_delta: delta,
        opponent_delta: -delta,
      };
    }
    // self loss => opponent wins
    const delta = computeDelta(opponentRating, selfRating);
    return {
      self_before: selfRating,
      opponent_before: opponentRating,
      self_after: selfRating - delta,
      opponent_after: opponentRating + delta,
      self_delta: -delta,
      opponent_delta: delta,
    };
  }
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/lib/elo-preview.test.ts
  ```
  预期 `Test Files  1 passed`，所有 case 通过。
- [ ] commit：
  ```bash
  git add web/src/lib/elo-preview.ts web/src/lib/elo-preview.test.ts
  git commit -m "feat(web): add elo-preview lib matching backend ComputeDelta (TDD)"
  ```

### Task 15: components/RankBadge.tsx —— 段位徽章（含组件测试）

**Files:**
- Create: `web/src/components/RankBadge.tsx`
- Test: `web/src/components/RankBadge.test.tsx`

> 纯展示组件，可先写组件再写测试，但本组件必须配组件测试（验证颜色/文本）。基于 `lib/rank.ts` 的 `danOf/danLabel/danColor`。段 0 显示“未定级”灰色；其余显示“段 N”并用对应配色作为边框/文字色。

步骤：

- [ ] 写组件 `web/src/components/RankBadge.tsx`，完整内容：
  ```tsx
  import { danOf, danLabel, danColor } from "@/lib/rank";
  import { cn } from "@/lib/utils";

  interface RankBadgeProps {
    rating: number;
    className?: string;
  }

  export default function RankBadge({ rating, className }: RankBadgeProps) {
    const dan = danOf(rating);
    const color = danColor(dan);
    const label = danLabel(dan);
    return (
      <span
        data-testid="rank-badge"
        data-dan={dan}
        className={cn(
          "inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-semibold",
          className,
        )}
        style={{ color, borderColor: color }}
      >
        {label}
      </span>
    );
  }
  ```
- [ ] 写组件测试 `web/src/components/RankBadge.test.tsx`，完整内容（验证文本与颜色随分数变化）：
  ```tsx
  import { describe, it, expect } from "vitest";
  import { render, screen } from "@testing-library/react";
  import RankBadge from "@/components/RankBadge";

  describe("RankBadge", () => {
    it("shows 未定级 with gray color below floor", () => {
      render(<RankBadge rating={1000} />);
      const badge = screen.getByTestId("rank-badge");
      expect(badge).toHaveTextContent("未定级");
      expect(badge).toHaveAttribute("data-dan", "0");
      expect(badge.style.color).toBe("rgb(156, 163, 175)"); // #9ca3af
    });

    it("shows 段 3 blue for rating 1500", () => {
      render(<RankBadge rating={1500} />);
      const badge = screen.getByTestId("rank-badge");
      expect(badge).toHaveTextContent("段 3");
      expect(badge).toHaveAttribute("data-dan", "3");
      expect(badge.style.color).toBe("rgb(74, 158, 255)"); // #4a9eff
    });

    it("shows 段 9 red for very high rating", () => {
      render(<RankBadge rating={2700} />);
      const badge = screen.getByTestId("rank-badge");
      expect(badge).toHaveTextContent("段 9");
      expect(badge).toHaveAttribute("data-dan", "9");
      expect(badge.style.color).toBe("rgb(240, 128, 128)"); // #f08080
    });
  });
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/components/RankBadge.test.tsx
  ```
  预期 `Test Files  1 passed`，3 个 case 通过。
- [ ] commit：
  ```bash
  git add web/src/components/RankBadge.tsx web/src/components/RankBadge.test.tsx
  git commit -m "feat(web): add RankBadge component with color/text tests"
  ```

### Task 16: lib/echarts-theme.ts + components/RatingChart.tsx —— 单玩家曲线

**Files:**
- Create: `web/src/lib/echarts-theme.ts`
- Create: `web/src/components/RatingChart.tsx`
- Test: `web/src/components/RatingChart.test.tsx`

> 纯展示组件，先写组件再写测试。`RatingChart` 单线 + 段位水平线（markLine：每个段位下边界画水平参考线，并以段位标签命名）。暗色主题。用 `echarts-for-react` 的 `ReactECharts`。测试用 mock `echarts-for-react` 验证传入的 `option` 含一条 line series 与 markLine。

步骤：

- [ ] 写 `web/src/lib/echarts-theme.ts`，完整内容（共享暗色配色与 5 色板）：
  ```ts
  export const ECHARTS_PALETTE = [
    "#4a9eff",
    "#7fd6a3",
    "#8b5cf6",
    "#e0c47d",
    "#f08080",
  ];

  export const AXIS_LINE_COLOR = "#3f3f46";
  export const AXIS_LABEL_COLOR = "#a1a1aa";
  export const SPLIT_LINE_COLOR = "#27272a";

  /** 段位下边界（与 rank.ts 区间一致），用于曲线段位水平参考线 */
  export const DAN_BOUNDARIES: { value: number; label: string }[] = [
    { value: 1050, label: "段 1" },
    { value: 1200, label: "段 2" },
    { value: 1400, label: "段 3" },
    { value: 1600, label: "段 4" },
    { value: 1800, label: "段 5" },
    { value: 2000, label: "段 6" },
    { value: 2200, label: "段 7" },
    { value: 2400, label: "段 8" },
    { value: 2600, label: "段 9" },
  ];
  ```
- [ ] 写组件 `web/src/components/RatingChart.tsx`，完整内容：
  ```tsx
  import ReactECharts from "echarts-for-react";
  import type { EChartsOption } from "echarts";
  import type { HistoryPoint } from "@/api/types";
  import {
    ECHARTS_PALETTE,
    AXIS_LABEL_COLOR,
    AXIS_LINE_COLOR,
    SPLIT_LINE_COLOR,
    DAN_BOUNDARIES,
  } from "@/lib/echarts-theme";

  interface RatingChartProps {
    points: HistoryPoint[];
    height?: number;
  }

  export default function RatingChart({ points, height = 420 }: RatingChartProps) {
    const data = points.map((p) => [p.played_at, p.rating] as [string, number]);

    const option: EChartsOption = {
      backgroundColor: "transparent",
      color: ECHARTS_PALETTE,
      grid: { left: 48, right: 24, top: 24, bottom: 40 },
      tooltip: {
        trigger: "axis",
        backgroundColor: "#18181b",
        borderColor: "#3f3f46",
        textStyle: { color: "#fafafa" },
      },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: AXIS_LINE_COLOR } },
        axisLabel: { color: AXIS_LABEL_COLOR },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        scale: true,
        axisLine: { lineStyle: { color: AXIS_LINE_COLOR } },
        axisLabel: { color: AXIS_LABEL_COLOR },
        splitLine: { lineStyle: { color: SPLIT_LINE_COLOR } },
      },
      series: [
        {
          name: "等级分",
          type: "line",
          smooth: true,
          showSymbol: true,
          symbolSize: 5,
          data,
          lineStyle: { width: 2 },
          markLine: {
            silent: true,
            symbol: "none",
            lineStyle: { color: SPLIT_LINE_COLOR, type: "dashed" },
            label: {
              color: AXIS_LABEL_COLOR,
              formatter: "{b}",
              position: "insideEndTop",
            },
            data: DAN_BOUNDARIES.map((b) => ({ yAxis: b.value, name: b.label })),
          },
        },
      ],
    };

    return (
      <div data-testid="rating-chart">
        <ReactECharts
          option={option}
          style={{ height, width: "100%" }}
          notMerge
        />
      </div>
    );
  }
  ```
- [ ] 写组件测试 `web/src/components/RatingChart.test.tsx`，完整内容（mock `echarts-for-react`，断言 option 结构）：
  ```tsx
  import { describe, it, expect, vi } from "vitest";
  import { render, screen } from "@testing-library/react";

  const optionSpy = vi.fn();
  vi.mock("echarts-for-react", () => ({
    default: (props: { option: unknown }) => {
      optionSpy(props.option);
      return <div data-testid="echarts-mock" />;
    },
  }));

  import RatingChart from "@/components/RatingChart";

  describe("RatingChart", () => {
    it("renders a single line series with dan markLine", () => {
      render(
        <RatingChart
          points={[
            { played_at: "2026-06-01T00:00:00Z", rating: 1500 },
            { played_at: "2026-06-02T00:00:00Z", rating: 1516 },
          ]}
        />,
      );
      expect(screen.getByTestId("rating-chart")).toBeInTheDocument();
      const option = optionSpy.mock.calls.at(-1)?.[0] as {
        series: { type: string; data: unknown[]; markLine: { data: unknown[] } }[];
      };
      expect(option.series).toHaveLength(1);
      expect(option.series[0].type).toBe("line");
      expect(option.series[0].data).toHaveLength(2);
      expect(option.series[0].markLine.data.length).toBe(9);
    });
  });
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/components/RatingChart.test.tsx
  ```
  预期 `Test Files  1 passed`，1 个 case 通过。
- [ ] commit：
  ```bash
  git add web/src/lib/echarts-theme.ts web/src/components/RatingChart.tsx web/src/components/RatingChart.test.tsx
  git commit -m "feat(web): add RatingChart with dan reference lines"
  ```

### Task 17: components/CompareChart.tsx —— 多玩家曲线对比

**Files:**
- Create: `web/src/components/CompareChart.tsx`
- Test: `web/src/components/CompareChart.test.tsx`

> 纯展示组件，先写组件再写测试。多线叠加，每条用后端 `CompareSeries.color`（若缺省回退到 5 色板按序取色）；`tooltip.trigger: "axis"` 即为“同步 tooltip”（一条竖线同时显示所有系列在该点的值），`axisPointer` 跨系列同步。暗色主题。

步骤：

- [ ] 写组件 `web/src/components/CompareChart.tsx`，完整内容：
  ```tsx
  import ReactECharts from "echarts-for-react";
  import type { EChartsOption } from "echarts";
  import type { CompareSeries } from "@/api/types";
  import {
    ECHARTS_PALETTE,
    AXIS_LABEL_COLOR,
    AXIS_LINE_COLOR,
    SPLIT_LINE_COLOR,
  } from "@/lib/echarts-theme";

  interface CompareChartProps {
    series: CompareSeries[];
    height?: number;
  }

  export default function CompareChart({ series, height = 480 }: CompareChartProps) {
    const echartsSeries = series.map((s, i) => ({
      name: s.username,
      type: "line" as const,
      smooth: true,
      showSymbol: false,
      data: s.points.map(
        (p) => [p.played_at, p.rating] as [string, number],
      ),
      lineStyle: {
        width: 2,
        color: s.color || ECHARTS_PALETTE[i % ECHARTS_PALETTE.length],
      },
      itemStyle: {
        color: s.color || ECHARTS_PALETTE[i % ECHARTS_PALETTE.length],
      },
    }));

    const option: EChartsOption = {
      backgroundColor: "transparent",
      legend: {
        data: series.map((s) => s.username),
        textStyle: { color: AXIS_LABEL_COLOR },
        top: 0,
      },
      grid: { left: 48, right: 24, top: 40, bottom: 40 },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "line", snap: true },
        backgroundColor: "#18181b",
        borderColor: "#3f3f46",
        textStyle: { color: "#fafafa" },
      },
      xAxis: {
        type: "time",
        axisLine: { lineStyle: { color: AXIS_LINE_COLOR } },
        axisLabel: { color: AXIS_LABEL_COLOR },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        scale: true,
        axisLine: { lineStyle: { color: AXIS_LINE_COLOR } },
        axisLabel: { color: AXIS_LABEL_COLOR },
        splitLine: { lineStyle: { color: SPLIT_LINE_COLOR } },
      },
      series: echartsSeries,
    };

    return (
      <div data-testid="compare-chart">
        <ReactECharts
          option={option}
          style={{ height, width: "100%" }}
          notMerge
        />
      </div>
    );
  }
  ```
- [ ] 写组件测试 `web/src/components/CompareChart.test.tsx`，完整内容（mock echarts，断言系列数量、轴触发 tooltip、颜色回退）：
  ```tsx
  import { describe, it, expect, vi } from "vitest";
  import { render, screen } from "@testing-library/react";

  const optionSpy = vi.fn();
  vi.mock("echarts-for-react", () => ({
    default: (props: { option: unknown }) => {
      optionSpy(props.option);
      return <div data-testid="echarts-mock" />;
    },
  }));

  import CompareChart from "@/components/CompareChart";

  describe("CompareChart", () => {
    it("renders one line per series with axis-trigger tooltip", () => {
      render(
        <CompareChart
          series={[
            {
              username: "alice",
              color: "#4a9eff",
              points: [{ played_at: "2026-06-01T00:00:00Z", rating: 1500 }],
            },
            {
              username: "bob",
              color: "",
              points: [{ played_at: "2026-06-01T00:00:00Z", rating: 1480 }],
            },
          ]}
        />,
      );
      expect(screen.getByTestId("compare-chart")).toBeInTheDocument();
      const option = optionSpy.mock.calls.at(-1)?.[0] as {
        series: { lineStyle: { color: string } }[];
        tooltip: { trigger: string };
      };
      expect(option.series).toHaveLength(2);
      expect(option.tooltip.trigger).toBe("axis");
      // bob has empty color -> falls back to palette[1]
      expect(option.series[1].lineStyle.color).toBe("#7fd6a3");
    });
  });
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/components/CompareChart.test.tsx
  ```
  预期 `Test Files  1 passed`，1 个 case 通过。
- [ ] commit：
  ```bash
  git add web/src/components/CompareChart.tsx web/src/components/CompareChart.test.tsx
  git commit -m "feat(web): add CompareChart with synced axis tooltip"
  ```

### Task 18: components/MatchTable.tsx —— 对局列表表格

**Files:**
- Create: `web/src/components/MatchTable.tsx`

> 纯展示组件。展示 `MatchView[]`：对手、结果（胜/负，胜绿负红）、赛前→赛后、delta（正绿负红带符号）、时间（本地化）。可选 `onDelete` 回调（管理员场景传入，渲染删除按钮）。

步骤：

- [ ] 写组件 `web/src/components/MatchTable.tsx`，完整内容：
  ```tsx
  import type { MatchView } from "@/api/types";
  import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
  } from "@/components/ui/table";
  import { Button } from "@/components/ui/button";
  import { cn } from "@/lib/utils";

  interface MatchTableProps {
    matches: MatchView[];
    onDelete?: (id: number) => void;
    onRestore?: (id: number) => void;
  }

  function formatTime(iso: string): string {
    const d = new Date(iso);
    return d.toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  export default function MatchTable({
    matches,
    onDelete,
    onRestore,
  }: MatchTableProps) {
    if (matches.length === 0) {
      return (
        <div className="py-8 text-center text-sm text-muted-foreground">
          暂无对局记录
        </div>
      );
    }
    return (
      <Table data-testid="match-table">
        <TableHeader>
          <TableRow>
            <TableHead>对手</TableHead>
            <TableHead>结果</TableHead>
            <TableHead className="text-right">赛前</TableHead>
            <TableHead className="text-right">赛后</TableHead>
            <TableHead className="text-right">变化</TableHead>
            <TableHead>时间</TableHead>
            {(onDelete || onRestore) && <TableHead className="text-right">操作</TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {matches.map((m) => (
            <TableRow key={m.id}>
              <TableCell className="font-medium">{m.opponent}</TableCell>
              <TableCell>
                <span
                  className={cn(
                    "font-semibold",
                    m.result === "win" ? "text-emerald-400" : "text-rose-400",
                  )}
                >
                  {m.result === "win" ? "胜" : "负"}
                </span>
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {m.rating_before}
              </TableCell>
              <TableCell className="text-right tabular-nums">
                {m.rating_after}
              </TableCell>
              <TableCell
                className={cn(
                  "text-right tabular-nums font-semibold",
                  m.delta >= 0 ? "text-emerald-400" : "text-rose-400",
                )}
              >
                {m.delta >= 0 ? `+${m.delta}` : m.delta}
              </TableCell>
              <TableCell className="text-muted-foreground">
                {formatTime(m.played_at)}
              </TableCell>
              {(onDelete || onRestore) && (
                <TableCell className="text-right">
                  {onDelete && (
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => onDelete(m.id)}
                    >
                      删除
                    </Button>
                  )}
                  {onRestore && (
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => onRestore(m.id)}
                    >
                      恢复
                    </Button>
                  )}
                </TableCell>
              )}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/components/MatchTable.tsx
  git commit -m "feat(web): add MatchTable component"
  ```

### Task 19: components/PlayerCombobox.tsx —— 玩家自动补全选择器

**Files:**
- Create: `web/src/components/PlayerCombobox.tsx`

> 纯展示/交互组件。用 shadcn 的 `Popover` + `Command` 实现自动补全。从 `/api/players` 拉全量玩家列表（React Query，`staleTime: 30s`），按用户名过滤。受控：`value`（用户名）+ `onChange`。可选 `exclude`（排除某些用户名，例如录入时排除自己）。

步骤：

- [ ] 写组件 `web/src/components/PlayerCombobox.tsx`，完整内容：
  ```tsx
  import { useState } from "react";
  import { useQuery } from "@tanstack/react-query";
  import { Check, ChevronsUpDown } from "lucide-react";
  import { listPlayers } from "@/api/players";
  import { Button } from "@/components/ui/button";
  import {
    Popover,
    PopoverContent,
    PopoverTrigger,
  } from "@/components/ui/popover";
  import {
    Command,
    CommandEmpty,
    CommandGroup,
    CommandInput,
    CommandItem,
    CommandList,
  } from "@/components/ui/command";
  import { cn } from "@/lib/utils";

  interface PlayerComboboxProps {
    value: string;
    onChange: (username: string) => void;
    exclude?: string[];
    placeholder?: string;
  }

  export default function PlayerCombobox({
    value,
    onChange,
    exclude = [],
    placeholder = "选择对手…",
  }: PlayerComboboxProps) {
    const [open, setOpen] = useState(false);
    const { data: players = [] } = useQuery({
      queryKey: ["players"],
      queryFn: listPlayers,
      staleTime: 30_000,
    });

    const options = players.filter((p) => !exclude.includes(p.username));

    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className="w-full justify-between"
            data-testid="player-combobox-trigger"
          >
            {value || placeholder}
            <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
          <Command>
            <CommandInput placeholder="搜索玩家…" />
            <CommandList>
              <CommandEmpty>未找到玩家</CommandEmpty>
              <CommandGroup>
                {options.map((p) => (
                  <CommandItem
                    key={p.id}
                    value={p.username}
                    onSelect={(selected) => {
                      onChange(selected);
                      setOpen(false);
                    }}
                  >
                    <Check
                      className={cn(
                        "mr-2 h-4 w-4",
                        value === p.username ? "opacity-100" : "opacity-0",
                      )}
                    />
                    {p.username}
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/components/PlayerCombobox.tsx
  git commit -m "feat(web): add PlayerCombobox autocomplete"
  ```

### Task 20: components/SubmitMatchDialog.tsx —— 录入弹窗（含 Elo 预览，配组件测试）

**Files:**
- Create: `web/src/components/SubmitMatchDialog.tsx`
- Test: `web/src/components/SubmitMatchDialog.test.tsx`

> 必须配组件测试（预览正确 + 提交触发 API）。布局 A 标准表单：对手（PlayerCombobox）→ 结果（胜/负）→ 时间（可选，默认当前）→ Elo 预览（用 `lib/elo-preview.ts` 的 `previewMatch`，基于自己当前 rating 与对手 rating）→ 提交（调用 `recordMatch`，成功后 toast + `invalidateQueries(['leaderboard','me','players'])`）。
> 表单用 react-hook-form + zod。对手 rating 从已加载的 players 列表里按用户名查。

步骤：

- [ ] 写组件 `web/src/components/SubmitMatchDialog.tsx`，完整内容：
  ```tsx
  import { useState } from "react";
  import { useForm } from "react-hook-form";
  import { zodResolver } from "@hookform/resolvers/zod";
  import { z } from "zod";
  import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
  import { toast } from "sonner";
  import {
    Dialog,
    DialogContent,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
  } from "@/components/ui/dialog";
  import { Button } from "@/components/ui/button";
  import { Label } from "@/components/ui/label";
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from "@/components/ui/select";
  import PlayerCombobox from "@/components/PlayerCombobox";
  import { listPlayers, getMe } from "@/api/players";
  import { recordMatch } from "@/api/matches";
  import { previewMatch } from "@/lib/elo-preview";
  import { ApiError } from "@/api/client";
  import type { MatchResult } from "@/api/types";

  const schema = z.object({
    opponent_username: z.string().min(1, "请选择对手"),
    result: z.enum(["win", "loss"]),
  });
  type FormValues = z.infer<typeof schema>;

  interface SubmitMatchDialogProps {
    trigger?: React.ReactNode;
  }

  export default function SubmitMatchDialog({ trigger }: SubmitMatchDialogProps) {
    const [open, setOpen] = useState(false);
    const qc = useQueryClient();

    const { data: me } = useQuery({ queryKey: ["me"], queryFn: getMe });
    const { data: players = [] } = useQuery({
      queryKey: ["players"],
      queryFn: listPlayers,
      staleTime: 30_000,
    });

    const {
      register,
      handleSubmit,
      watch,
      setValue,
      reset,
      formState: { errors },
    } = useForm<FormValues>({
      resolver: zodResolver(schema),
      defaultValues: { opponent_username: "", result: "win" },
    });

    const opponentUsername = watch("opponent_username");
    const result = watch("result");
    const opponent = players.find((p) => p.username === opponentUsername);

    const preview =
      me && opponent
        ? previewMatch(me.rating, opponent.rating, result)
        : null;

    const mutation = useMutation({
      mutationFn: (values: FormValues) =>
        recordMatch({
          opponent_username: values.opponent_username,
          result: values.result,
        }),
      onSuccess: (res) => {
        toast.success(
          `已录入：你 ${res.winner_delta >= 0 ? "+" : ""}${
            res.new_self_rating
          }`,
        );
        qc.invalidateQueries({ queryKey: ["leaderboard"] });
        qc.invalidateQueries({ queryKey: ["me"] });
        qc.invalidateQueries({ queryKey: ["players"] });
        reset();
        setOpen(false);
      },
      onError: (err) => {
        const code = err instanceof ApiError ? err.code : "UNKNOWN";
        const msg =
          code === "SELF_MATCH"
            ? "不能和自己对局"
            : code === "PLAYER_NOT_FOUND"
              ? "对手不存在"
              : "录入失败";
        toast.error(msg);
      },
    });

    return (
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          {trigger ?? <Button data-testid="open-submit">录入对局</Button>}
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>录入对局</DialogTitle>
          </DialogHeader>
          <form
            onSubmit={handleSubmit((v) => mutation.mutate(v))}
            className="space-y-4"
          >
            <div className="space-y-2">
              <Label>对手</Label>
              <PlayerCombobox
                value={opponentUsername}
                onChange={(u) =>
                  setValue("opponent_username", u, { shouldValidate: true })
                }
                exclude={me ? [me.username] : []}
              />
              {errors.opponent_username && (
                <p className="text-xs text-rose-400">
                  {errors.opponent_username.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label>结果</Label>
              <Select
                value={result}
                onValueChange={(v) =>
                  setValue("result", v as MatchResult, { shouldValidate: true })
                }
              >
                <SelectTrigger data-testid="result-select">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="win">我赢了</SelectItem>
                  <SelectItem value="loss">我输了</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {preview && (
              <div
                data-testid="elo-preview"
                className="rounded-md border bg-muted/40 p-3 text-sm"
              >
                <div className="flex justify-between">
                  <span>我：{preview.self_before} →</span>
                  <span
                    className={
                      preview.self_delta >= 0
                        ? "text-emerald-400"
                        : "text-rose-400"
                    }
                  >
                    {preview.self_after}（
                    {preview.self_delta >= 0 ? "+" : ""}
                    {preview.self_delta}）
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>{opponent?.username}：{preview.opponent_before} →</span>
                  <span
                    className={
                      preview.opponent_delta >= 0
                        ? "text-emerald-400"
                        : "text-rose-400"
                    }
                  >
                    {preview.opponent_after}（
                    {preview.opponent_delta >= 0 ? "+" : ""}
                    {preview.opponent_delta}）
                  </span>
                </div>
              </div>
            )}

            <input type="hidden" {...register("opponent_username")} />

            <DialogFooter>
              <Button
                type="submit"
                disabled={mutation.isPending || !opponent}
                data-testid="submit-match"
              >
                {mutation.isPending ? "提交中…" : "提交"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    );
  }
  ```
- [ ] 写组件测试 `web/src/components/SubmitMatchDialog.test.tsx`，完整内容（mock api，打开弹窗→选对手→看到预览→提交触发 recordMatch）：
  ```tsx
  import { describe, it, expect, vi, beforeEach } from "vitest";
  import { render, screen, waitFor } from "@testing-library/react";
  import userEvent from "@testing-library/user-event";
  import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
  import SubmitMatchDialog from "@/components/SubmitMatchDialog";
  import * as playersApi from "@/api/players";
  import * as matchesApi from "@/api/matches";

  function setup() {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    return render(
      <QueryClientProvider client={qc}>
        <SubmitMatchDialog />
      </QueryClientProvider>,
    );
  }

  describe("SubmitMatchDialog", () => {
    beforeEach(() => {
      vi.restoreAllMocks();
      vi.spyOn(playersApi, "getMe").mockResolvedValue({
        id: 1,
        username: "alice",
        rating: 1500,
        dan: 3,
        created_at: "2026-06-25T00:00:00Z",
      });
      vi.spyOn(playersApi, "listPlayers").mockResolvedValue([
        { id: 1, username: "alice", rating: 1500, dan: 3, created_at: "2026-06-25T00:00:00Z" },
        { id: 2, username: "bob", rating: 1500, dan: 3, created_at: "2026-06-25T00:00:00Z" },
      ]);
    });

    it("shows elo preview and submits via recordMatch", async () => {
      const recordSpy = vi
        .spyOn(matchesApi, "recordMatch")
        .mockResolvedValue({
          id: 10,
          winner_delta: 8,
          loser_delta: -8,
          new_self_rating: 1508,
          new_opponent_rating: 1492,
        });
      const user = userEvent.setup();
      setup();

      await user.click(await screen.findByTestId("open-submit"));
      await user.click(await screen.findByTestId("player-combobox-trigger"));
      await user.click(await screen.findByText("bob"));

      const preview = await screen.findByTestId("elo-preview");
      expect(preview).toHaveTextContent("1508");
      expect(preview).toHaveTextContent("+8");
      expect(preview).toHaveTextContent("1492");

      await user.click(screen.getByTestId("submit-match"));

      await waitFor(() => {
        expect(recordSpy).toHaveBeenCalledWith({
          opponent_username: "bob",
          result: "win",
        });
      });
    });
  });
  ```
- [ ] 运行测试确认通过：
  ```bash
  pnpm test src/components/SubmitMatchDialog.test.tsx
  ```
  预期 `Test Files  1 passed`，1 个 case 通过。
- [ ] commit：
  ```bash
  git add web/src/components/SubmitMatchDialog.tsx web/src/components/SubmitMatchDialog.test.tsx
  git commit -m "feat(web): add SubmitMatchDialog with elo preview and submit tests"
  ```

### Task 21: components/Layout.tsx —— 顶部导航 + 内容容器

**Files:**
- Create: `web/src/components/Layout.tsx`

> 纯展示组件。spec §7.3 顶部导航：左 `🎯 go_ultra`，中部链接“我的 / 排行榜 / 对比 / 录入对局”，右上用户菜单（个人主页 / 登出 / 管理面板若可用）。导航链接用 react-router `NavLink`，激活态高亮。“录入对局”点击弹出 `SubmitMatchDialog`。登出调用 `logout` 后跳 `/login`。

步骤：

- [ ] 写组件 `web/src/components/Layout.tsx`，完整内容：
  ```tsx
  import type { ReactNode } from "react";
  import { NavLink, useNavigate } from "react-router-dom";
  import { useQueryClient } from "@tanstack/react-query";
  import { ChevronDown } from "lucide-react";
  import { useAuth } from "@/hooks/useAuth";
  import { logout } from "@/api/players";
  import SubmitMatchDialog from "@/components/SubmitMatchDialog";
  import { Button } from "@/components/ui/button";
  import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
  } from "@/components/ui/dropdown-menu";
  import { cn } from "@/lib/utils";

  const navItems = [
    { to: "/me", label: "我的" },
    { to: "/leaderboard", label: "排行榜" },
    { to: "/compare", label: "对比" },
  ];

  export default function Layout({ children }: { children: ReactNode }) {
    const { player } = useAuth();
    const navigate = useNavigate();
    const qc = useQueryClient();

    async function handleLogout() {
      await logout();
      qc.clear();
      navigate("/login", { replace: true });
    }

    return (
      <div className="min-h-screen bg-background">
        <header className="border-b">
          <div className="container flex h-14 items-center justify-between">
            <div className="flex items-center gap-6">
              <NavLink to="/me" className="text-lg font-bold">
                🎯 go_ultra
              </NavLink>
              <nav className="flex items-center gap-1">
                {navItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) =>
                      cn(
                        "rounded-md px-3 py-1.5 text-sm transition-colors",
                        isActive
                          ? "bg-secondary text-secondary-foreground"
                          : "text-muted-foreground hover:text-foreground",
                      )
                    }
                  >
                    {item.label}
                  </NavLink>
                ))}
                <SubmitMatchDialog
                  trigger={
                    <Button variant="ghost" size="sm" className="text-sm">
                      录入对局
                    </Button>
                  }
                />
              </nav>
            </div>

            {player && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="gap-1">
                    {player.username}
                    <ChevronDown className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => navigate("/me")}>
                    个人主页
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => navigate("/admin")}>
                    管理面板
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={handleLogout}>登出</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
          </div>
        </header>
        <main className="container py-6">{children}</main>
      </div>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/components/Layout.tsx
  git commit -m "feat(web): add Layout with top nav and user menu"
  ```

### Task 22: pages/Login.tsx —— 登录页（仅用户名）

**Files:**
- Modify: `web/src/pages/Login.tsx`

> spec §2 仅用户名登录（隐式注册）。表单用 react-hook-form + zod（用户名 3-32，trim 后非空）。成功后 `invalidateQueries(['me'])` 并跳 `/me`。

步骤：

- [ ] 用以下完整内容覆盖 `web/src/pages/Login.tsx`：
  ```tsx
  import { useForm } from "react-hook-form";
  import { zodResolver } from "@hookform/resolvers/zod";
  import { z } from "zod";
  import { useNavigate } from "react-router-dom";
  import { useMutation, useQueryClient } from "@tanstack/react-query";
  import { toast } from "sonner";
  import { login } from "@/api/players";
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
  } from "@/components/ui/card";
  import { ApiError } from "@/api/client";

  const schema = z.object({
    username: z
      .string()
      .trim()
      .min(3, "用户名至少 3 个字符")
      .max(32, "用户名至多 32 个字符"),
  });
  type FormValues = z.infer<typeof schema>;

  export default function Login() {
    const navigate = useNavigate();
    const qc = useQueryClient();
    const {
      register,
      handleSubmit,
      formState: { errors },
    } = useForm<FormValues>({ resolver: zodResolver(schema) });

    const mutation = useMutation({
      mutationFn: (values: FormValues) => login(values.username),
      onSuccess: async () => {
        await qc.invalidateQueries({ queryKey: ["me"] });
        navigate("/me", { replace: true });
      },
      onError: (err) => {
        const msg = err instanceof ApiError ? err.message : "登录失败";
        toast.error(msg);
      },
    });

    return (
      <div className="flex min-h-screen items-center justify-center">
        <Card className="w-96">
          <CardHeader>
            <CardTitle className="text-center text-2xl">🎯 go_ultra</CardTitle>
          </CardHeader>
          <CardContent>
            <form
              onSubmit={handleSubmit((v) => mutation.mutate(v))}
              className="space-y-4"
            >
              <div className="space-y-2">
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  autoComplete="username"
                  placeholder="输入用户名，不存在将自动创建"
                  {...register("username")}
                />
                {errors.username && (
                  <p className="text-xs text-rose-400">
                    {errors.username.message}
                  </p>
                )}
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={mutation.isPending}
              >
                {mutation.isPending ? "登录中…" : "登录"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/pages/Login.tsx
  git commit -m "feat(web): implement Login page"
  ```

### Task 23: components/PlayerOverview.tsx —— 主页/详情共用布局 B

**Files:**
- Create: `web/src/components/PlayerOverview.tsx`

> spec §7.2：我的主页布局 B（大曲线 + 右侧栏统计 + 录入按钮 + 最近对局）；玩家详情复用此布局并在右上加“📊 对比”按钮。把这块抽成 `PlayerOverview`，由 `Dashboard` 与 `PlayerDetail` 复用。
> 参数：`username`（要展示谁）、`isSelf`（是否本人，决定是否显示录入按钮）。内部用 React Query 拉 detail / history / matches，`staleTime: 30s`。history 前端 prepend 起点由后端 History 已处理（契约 `History` prepend (createdAt, DefaultRating)），前端直接画。

步骤：

- [ ] 写组件 `web/src/components/PlayerOverview.tsx`，完整内容：
  ```tsx
  import { useQuery } from "@tanstack/react-query";
  import { useNavigate } from "react-router-dom";
  import {
    getPlayer,
    getPlayerHistory,
    getPlayerMatches,
  } from "@/api/players";
  import RatingChart from "@/components/RatingChart";
  import RankBadge from "@/components/RankBadge";
  import MatchTable from "@/components/MatchTable";
  import SubmitMatchDialog from "@/components/SubmitMatchDialog";
  import { Button } from "@/components/ui/button";
  import {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
  } from "@/components/ui/card";

  interface PlayerOverviewProps {
    username: string;
    isSelf: boolean;
  }

  export default function PlayerOverview({
    username,
    isSelf,
  }: PlayerOverviewProps) {
    const navigate = useNavigate();

    const detailQuery = useQuery({
      queryKey: ["player", username],
      queryFn: () => getPlayer(username),
      staleTime: 30_000,
    });
    const historyQuery = useQuery({
      queryKey: ["player-history", username],
      queryFn: () => getPlayerHistory(username),
      staleTime: 30_000,
    });
    const matchesQuery = useQuery({
      queryKey: ["player-matches", username],
      queryFn: () => getPlayerMatches(username, { limit: 20, offset: 0 }),
      staleTime: 30_000,
    });

    const detail = detailQuery.data;

    return (
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_320px]">
        <section className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{username}</h1>
              {detail && <RankBadge rating={detail.rating} />}
            </div>
            {!isSelf && (
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  navigate(`/compare?p=${encodeURIComponent(username)}`)
                }
              >
                📊 对比
              </Button>
            )}
          </div>
          <Card>
            <CardContent className="pt-6">
              {historyQuery.data ? (
                <RatingChart points={historyQuery.data} />
              ) : (
                <div className="flex h-[420px] items-center justify-center text-muted-foreground">
                  加载中…
                </div>
              )}
            </CardContent>
          </Card>
        </section>

        <aside className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">统计</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              {detail ? (
                <>
                  <Row label="当前等级分" value={String(detail.rating)} />
                  <Row label="胜" value={String(detail.stats.wins)} />
                  <Row label="负" value={String(detail.stats.losses)} />
                  <Row
                    label="胜率"
                    value={`${(detail.stats.win_rate * 100).toFixed(1)}%`}
                  />
                  <Row
                    label="当前连胜"
                    value={String(detail.stats.current_streak)}
                  />
                  <Row
                    label="最长连胜"
                    value={String(detail.stats.longest_streak)}
                  />
                </>
              ) : (
                <div className="text-muted-foreground">加载中…</div>
              )}
            </CardContent>
          </Card>

          {isSelf && (
            <SubmitMatchDialog
              trigger={<Button className="w-full">录入对局</Button>}
            />
          )}

          <Card>
            <CardHeader>
              <CardTitle className="text-base">最近对局</CardTitle>
            </CardHeader>
            <CardContent>
              <MatchTable matches={matchesQuery.data ?? []} />
            </CardContent>
          </Card>
        </aside>
      </div>
    );
  }

  function Row({ label, value }: { label: string; value: string }) {
    return (
      <div className="flex justify-between">
        <span className="text-muted-foreground">{label}</span>
        <span className="font-medium tabular-nums">{value}</span>
      </div>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/components/PlayerOverview.tsx
  git commit -m "feat(web): add shared PlayerOverview layout B"
  ```

### Task 24: pages/Dashboard.tsx + pages/PlayerDetail.tsx —— 复用 PlayerOverview

**Files:**
- Modify: `web/src/pages/Dashboard.tsx`
- Modify: `web/src/pages/PlayerDetail.tsx`

步骤：

- [ ] 用以下完整内容覆盖 `web/src/pages/Dashboard.tsx`（自己的主页，`isSelf`）：
  ```tsx
  import Layout from "@/components/Layout";
  import PlayerOverview from "@/components/PlayerOverview";
  import { useAuth } from "@/hooks/useAuth";

  export default function Dashboard() {
    const { player } = useAuth();
    if (!player) {
      return (
        <Layout>
          <div className="text-muted-foreground">加载中…</div>
        </Layout>
      );
    }
    return (
      <Layout>
        <PlayerOverview username={player.username} isSelf />
      </Layout>
    );
  }
  ```
- [ ] 用以下完整内容覆盖 `web/src/pages/PlayerDetail.tsx`（任意玩家，复用布局；若 URL 用户名即本人则按本人处理）：
  ```tsx
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
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/pages/Dashboard.tsx web/src/pages/PlayerDetail.tsx
  git commit -m "feat(web): implement Dashboard and PlayerDetail via PlayerOverview"
  ```

### Task 25: pages/Leaderboard.tsx —— 排行榜（领奖台 + 表格）

**Files:**
- Modify: `web/src/pages/Leaderboard.tsx`

> spec §7.2 布局 A：Top 3 领奖台，第 4 名起紧凑表格。用 React Query 拉 `/api/leaderboard`，`staleTime: 30s`。每行用户名可点击跳 `/players/:username`。段位用 `RankBadge`（基于 rating）。

步骤：

- [ ] 用以下完整内容覆盖 `web/src/pages/Leaderboard.tsx`：
  ```tsx
  import { useQuery } from "@tanstack/react-query";
  import { useNavigate } from "react-router-dom";
  import Layout from "@/components/Layout";
  import RankBadge from "@/components/RankBadge";
  import { getLeaderboard } from "@/api/matches";
  import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
  } from "@/components/ui/table";
  import { Card, CardContent } from "@/components/ui/card";
  import { cn } from "@/lib/utils";
  import type { LeaderboardRow } from "@/api/types";

  const PODIUM_STYLE = [
    "border-yellow-500/60",
    "border-zinc-400/60",
    "border-amber-700/60",
  ];

  export default function Leaderboard() {
    const navigate = useNavigate();
    const { data: rows = [] } = useQuery({
      queryKey: ["leaderboard"],
      queryFn: () => getLeaderboard(0),
      staleTime: 30_000,
    });

    const top3 = rows.slice(0, 3);
    const rest = rows.slice(3);

    return (
      <Layout>
        <h1 className="mb-6 text-2xl font-bold">排行榜</h1>

        {top3.length > 0 && (
          <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-3">
            {top3.map((row, i) => (
              <PodiumCard
                key={row.username}
                row={row}
                rankClass={PODIUM_STYLE[i]}
                onClick={() =>
                  navigate(`/players/${encodeURIComponent(row.username)}`)
                }
              />
            ))}
          </div>
        )}

        {rest.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">名次</TableHead>
                <TableHead>玩家</TableHead>
                <TableHead>段位</TableHead>
                <TableHead className="text-right">等级分</TableHead>
                <TableHead className="text-right">局数</TableHead>
                <TableHead className="text-right">胜率</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rest.map((row) => (
                <TableRow
                  key={row.username}
                  className="cursor-pointer"
                  onClick={() =>
                    navigate(`/players/${encodeURIComponent(row.username)}`)
                  }
                >
                  <TableCell className="tabular-nums">{row.rank}</TableCell>
                  <TableCell className="font-medium">{row.username}</TableCell>
                  <TableCell>
                    <RankBadge rating={row.rating} />
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {row.rating}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {row.games_played}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {(row.win_rate * 100).toFixed(0)}%
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {rows.length === 0 && (
          <div className="py-12 text-center text-muted-foreground">
            暂无排名数据
          </div>
        )}
      </Layout>
    );
  }

  function PodiumCard({
    row,
    rankClass,
    onClick,
  }: {
    row: LeaderboardRow;
    rankClass: string;
    onClick: () => void;
  }) {
    return (
      <Card
        className={cn("cursor-pointer border-2", rankClass)}
        onClick={onClick}
      >
        <CardContent className="flex flex-col items-center gap-2 py-6">
          <div className="text-3xl font-bold tabular-nums">#{row.rank}</div>
          <div className="text-lg font-semibold">{row.username}</div>
          <RankBadge rating={row.rating} />
          <div className="text-2xl font-bold tabular-nums">{row.rating}</div>
          <div className="text-xs text-muted-foreground">
            {row.games_played} 局 · 胜率 {(row.win_rate * 100).toFixed(0)}%
          </div>
        </CardContent>
      </Card>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/pages/Leaderboard.tsx
  git commit -m "feat(web): implement Leaderboard with podium and table"
  ```

### Task 26: pages/Compare.tsx —— 多人对比（左侧栏 + 右大曲线 + 下方头对头卡）

**Files:**
- Modify: `web/src/pages/Compare.tsx`

> spec §7.2 B+C 合并：左侧栏用 `PlayerCombobox` 增删玩家（上限 10，与后端一致）+ 显示已选列表（可移除）；右侧 `CompareChart`；下方头对头卡片网格。已选玩家通过 URL query `?p=a,b,c` 同步（刷新可恢复）。用 React Query 拉 `/api/compare`，`staleTime: 30s`。

步骤：

- [ ] 用以下完整内容覆盖 `web/src/pages/Compare.tsx`：
  ```tsx
  import { useMemo } from "react";
  import { useSearchParams } from "react-router-dom";
  import { useQuery } from "@tanstack/react-query";
  import { X } from "lucide-react";
  import Layout from "@/components/Layout";
  import CompareChart from "@/components/CompareChart";
  import PlayerCombobox from "@/components/PlayerCombobox";
  import { getCompare } from "@/api/matches";
  import { Button } from "@/components/ui/button";
  import {
    Card,
    CardContent,
    CardHeader,
    CardTitle,
  } from "@/components/ui/card";

  const MAX_PLAYERS = 10;

  export default function Compare() {
    const [searchParams, setSearchParams] = useSearchParams();

    const usernames = useMemo(() => {
      const p = searchParams.get("p");
      return p ? p.split(",").filter(Boolean) : [];
    }, [searchParams]);

    function setUsernames(next: string[]) {
      if (next.length === 0) {
        setSearchParams({});
      } else {
        setSearchParams({ p: next.join(",") });
      }
    }

    function addPlayer(username: string) {
      if (
        username &&
        !usernames.includes(username) &&
        usernames.length < MAX_PLAYERS
      ) {
        setUsernames([...usernames, username]);
      }
    }

    function removePlayer(username: string) {
      setUsernames(usernames.filter((u) => u !== username));
    }

    const { data } = useQuery({
      queryKey: ["compare", usernames],
      queryFn: () => getCompare(usernames),
      enabled: usernames.length >= 1,
      staleTime: 30_000,
    });

    return (
      <Layout>
        <h1 className="mb-6 text-2xl font-bold">多人对比</h1>
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
          <aside className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">选择玩家</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <PlayerCombobox
                  value=""
                  onChange={addPlayer}
                  exclude={usernames}
                  placeholder="添加玩家…"
                />
                <div className="space-y-2">
                  {usernames.map((u) => (
                    <div
                      key={u}
                      className="flex items-center justify-between rounded-md border px-3 py-1.5 text-sm"
                    >
                      <span>{u}</span>
                      <button
                        type="button"
                        aria-label={`移除 ${u}`}
                        onClick={() => removePlayer(u)}
                        className="text-muted-foreground hover:text-foreground"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                  {usernames.length === 0 && (
                    <p className="text-xs text-muted-foreground">
                      至少添加一名玩家
                    </p>
                  )}
                </div>
              </CardContent>
            </Card>
          </aside>

          <section className="space-y-6">
            <Card>
              <CardContent className="pt-6">
                {data && data.series.length > 0 ? (
                  <CompareChart series={data.series} />
                ) : (
                  <div className="flex h-[480px] items-center justify-center text-muted-foreground">
                    选择玩家以查看曲线
                  </div>
                )}
              </CardContent>
            </Card>

            {data && data.head_to_head.length > 0 && (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {data.head_to_head.map((h) => (
                  <Card key={`${h.a}-${h.b}`}>
                    <CardContent className="py-4">
                      <div className="mb-2 text-sm font-medium">
                        {h.a} vs {h.b}
                      </div>
                      <div className="flex items-baseline justify-between">
                        <span className="text-2xl font-bold tabular-nums text-emerald-400">
                          {h.a_wins}
                        </span>
                        <span className="text-muted-foreground">:</span>
                        <span className="text-2xl font-bold tabular-nums text-rose-400">
                          {h.b_wins}
                        </span>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </section>
        </div>
      </Layout>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/pages/Compare.tsx
  git commit -m "feat(web): implement Compare page with sidebar and head-to-head"
  ```

### Task 27: pages/Admin.tsx —— 管理员面板（已删除列表 + 恢复）

**Files:**
- Modify: `web/src/pages/Admin.tsx`

> spec §7.2：已删除对局列表（表格风格）可恢复。`AdminGuard`（阶段 5）已套在外层负责未授权时的密码框，本页假定已授权。用 React Query 拉 `/api/admin/matches/deleted`；恢复调用 `restoreMatch` 后 invalidate。复用 `MatchTable` 的 `onRestore`。

步骤：

- [ ] 用以下完整内容覆盖 `web/src/pages/Admin.tsx`：
  ```tsx
  import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
  import { toast } from "sonner";
  import Layout from "@/components/Layout";
  import MatchTable from "@/components/MatchTable";
  import { listDeletedMatches, restoreMatch } from "@/api/admin";
  import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

  export default function Admin() {
    const qc = useQueryClient();
    const { data: deleted = [] } = useQuery({
      queryKey: ["admin-deleted-matches"],
      queryFn: listDeletedMatches,
      staleTime: 10_000,
    });

    const restoreMutation = useMutation({
      mutationFn: (id: number) => restoreMatch(id),
      onSuccess: () => {
        toast.success("已恢复对局");
        qc.invalidateQueries({ queryKey: ["admin-deleted-matches"] });
        qc.invalidateQueries({ queryKey: ["leaderboard"] });
      },
      onError: () => toast.error("恢复失败"),
    });

    return (
      <Layout>
        <h1 className="mb-6 text-2xl font-bold">管理员面板</h1>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">已删除对局</CardTitle>
          </CardHeader>
          <CardContent>
            <MatchTable
              matches={deleted}
              onRestore={(id) => restoreMutation.mutate(id)}
            />
          </CardContent>
        </Card>
      </Layout>
    );
  }
  ```
- [ ] 验证类型可编译：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] commit：
  ```bash
  git add web/src/pages/Admin.tsx
  git commit -m "feat(web): implement Admin panel with deleted-match restore"
  ```

### Task 28: 全量前端校验（编译 + 全部测试 + 生产构建）

**Files:**
- 无新增文件（验证 + 修复门槛）

> 阶段收尾门槛：确保整个前端工程编译通过、全部 vitest 通过、生产构建成功。

步骤：

- [ ] 运行 TypeScript 全量类型检查：
  ```bash
  pnpm exec tsc -b
  ```
  预期无输出、退出码 0。
- [ ] 运行全部前端测试：
  ```bash
  pnpm test
  ```
  预期所有测试文件通过：`rank.test.ts`、`elo-preview.test.ts`、`client.test.ts`、`AuthGuard.test.tsx`、`RankBadge.test.tsx`、`RatingChart.test.tsx`、`CompareChart.test.tsx`、`SubmitMatchDialog.test.tsx` 全绿，输出 `Test Files  8 passed`。
- [ ] 运行覆盖率检查并确认 `lib/*` 达标（契约 §9.4：前端 `lib/*` ≥ 80%，`components/*` ≥ 50%）：
  ```bash
  pnpm test:coverage
  ```
  预期输出覆盖率表，`src/lib/rank.ts` 与 `src/lib/elo-preview.ts` 行覆盖 100%。
- [ ] 运行生产构建（确认 `vite build` 产出 `web/dist`，供 Caddy 静态服务）：
  ```bash
  pnpm build
  ```
  预期输出 `vite v5.x building for production...`、`✓ built in ...s`，生成 `web/dist/index.html` 与 `web/dist/assets/`。
- [ ] commit（构建产物不入库，仅记录可能的锁文件/配置微调；若无改动则跳过本步）：
  ```bash
  git add web/package.json web/pnpm-lock.yaml
  git commit -m "chore(web): verify build and full test suite pass"
  ```

---

## 阶段 7：部署与运维脚本

> 本阶段以脚本与配置文件为主，**不依赖前面阶段的代码即可单独完成 7.1–7.6 的脚本/配置编写**，但 `start.bat` 的健康探测依赖后端已实现 `GET /api/healthz`（契约 http 层第 219 行：返回 200 `{"status":"ok"}`），`reset-admin-password` 子命令（Task 7.7）依赖 domain/db/service 层已存在。配置文件类无单元测试，每个 Task 给出"如何手动验证"的确切命令与预期输出；脚本/代码类仍遵循 commit 步骤。所有路径以仓库根 `go_ultra/` 为基准。

---

### Task 7.1: 编写 Caddyfile（反代 + 速率限制 + SPA fallback + 日志）

**Files:**
- Create: `Caddyfile`
- Verify: 手动 `caddy validate`（无单元测试）

**背景与依赖说明（必读）：**

speс §8.3 给出的 Caddyfile 用了 `rate_limit` 指令。**这不是 Caddy 官方内置指令**，需要带 [`caddy-ratelimit`](https://github.com/mholt/caddy-ratelimit) 插件的 Caddy 二进制。官方下载的 `caddy.exe` 没有该指令，直接 `caddy validate` 会报 `unrecognized directive: rate_limit`。因此本 Task 必须先构建带插件的二进制，再写 Caddyfile，再验证。

步骤：

- [ ] **准备带插件的 caddy 二进制（两种方式任选其一，推荐方式 A）**

  方式 A — 用 `xcaddy` 本地构建（需要本机已安装 Go 1.22+）：
  ```bat
  go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
  xcaddy build --with github.com/mholt/caddy-ratelimit
  ```
  上述命令在当前目录生成 `caddy.exe`（已内置 ratelimit 插件）。把它放到 `PATH` 中，或放到仓库根 `go_ultra/caddy.exe`（注意：若放仓库根，请在 `.gitignore` 中忽略它，不提交二进制）。

  方式 B — 不想装 Go，用 Caddy 官方 Download 页在线定制下载：
  访问 `https://caddyserver.com/download`，平台选 `windows / amd64`，在 "add packages" 搜索框输入 `caddy-ratelimit` 并加入 `github.com/mholt/caddy-ratelimit`，点击 "Download" 得到带插件的 `caddy_windows_amd64_custom.exe`，重命名为 `caddy.exe` 并放入 `PATH`。

  验证插件已内置：
  ```bat
  caddy list-modules
  ```
  预期输出中包含一行：
  ```
  http.handlers.rate_limit
  ```
  若没有这一行，说明用的是无插件的官方二进制，回到方式 A 或 B 重新获取。

- [ ] **创建 `Caddyfile`（完整内容，可直接粘贴）**

  `caddy-ratelimit` 插件需要在全局 options 里启用对应的 order，否则 `rate_limit` 指令在 `handle` 外/内的执行顺序不确定。下面写法把 `rate_limit` 放进 `/api/*` 的 `handle` 块内，并显式声明指令顺序，确保 60 次/分钟 的限制只作用于 API：

  ```caddyfile
  {
  	order rate_limit before reverse_proxy
  }

  :443 {
  	tls internal

  	log {
  		output file E:/go_ultra/logs/caddy.log {
  			roll_size 10MiB
  			roll_keep 7
  		}
  		format json
  	}

  	handle /api/* {
  		rate_limit {
  			zone api_zone {
  				key {remote_host}
  				events 60
  				window 1m
  			}
  		}
  		reverse_proxy localhost:8080
  	}

  	handle {
  		root * E:/go_ultra/web/dist
  		try_files {path} /index.html
  		file_server
  	}
  }
  ```

  说明：
  - `tls internal`：本机自签证书；公网 TLS 由 Cloudflare 边缘签发，Tunnel 内部到 Caddy 这一段用内部证书即可。
  - `rate_limit` 的 `zone api_zone`：按 `{remote_host}`（客户端 IP）分桶，滑动窗口 `1m` 内最多 `60` 个事件 = spec §8.3 要求的"每 IP 60 req/min"；超出返回 HTTP 429（与 spec §6.2 的 `RATE_LIMITED` 对应）。
  - `/api/*` → `reverse_proxy localhost:8080`：转发到 Go 后端（spec §3.1 拓扑）。
  - `handle {}` 默认块：`root` 指向 `web/dist`，`try_files {path} /index.html` 实现 SPA history 路由 fallback，`file_server` 提供静态文件。
  - 日志：JSON 格式写到 `E:/go_ultra/logs/caddy.log`，单文件 10MiB 滚动、保留 7 个（与 spec §8.1 "保留 7 天"精神一致）。
  - 路径用绝对路径 `E:/go_ultra/...`：换部署机器时改这两处即可（spec §1.4 适配点）。

- [ ] **手动验证（配置类无单元测试，用 `caddy validate`）**

  先确保日志目录存在（`caddy validate` 不会创建目录，但语法校验本身不写日志，可不创建；为稳妥起见仍创建）：
  ```bat
  if not exist "E:\go_ultra\logs" mkdir "E:\go_ultra\logs"
  caddy validate --config Caddyfile
  ```
  预期输出（结尾应出现 `Valid configuration`，无 error）：
  ```
  ...
  INFO    using config from file  {"file": "Caddyfile"}
  INFO    adapted config to JSON  {"adapter": "caddyfile"}
  Valid configuration
  ```
  常见失败与排查：
  - `unrecognized directive: rate_limit` → 用的二进制没装插件，回到第一步重建。
  - `Error during parsing: ...` → 检查大括号配对与缩进（Caddyfile 用 Tab 缩进）。

  可选的运行期验证（确认 SPA fallback，无需后端）：
  ```bat
  caddy run --config Caddyfile
  ```
  另开一个终端：
  ```bat
  curl -k https://localhost/some/spa/route
  ```
  预期返回 `web/dist/index.html` 的 HTML 内容（HTTP 200），而非 404 —— 证明 `try_files ... /index.html` 生效。验证完按 `Ctrl+C` 停止。

- [ ] **commit**
  ```bat
  git add Caddyfile
  git commit -m "feat(deploy): add Caddyfile with api reverse proxy, 60r/m rate limit and SPA fallback"
  ```

---

### Task 7.2: 编写 scripts/dev.bat（并行 vite dev + go run）

**Files:**
- Create: `scripts/dev.bat`
- Verify: 手动运行（无单元测试）

步骤：

- [ ] **创建 `scripts/dev.bat`（完整内容，可直接粘贴）**

  用 `start` 在独立窗口里并行拉起前后端：前端 `pnpm dev`（vite，spec §3.3 监听 :5173 并 proxy `/api` 到 :8080），后端 `go run ./cmd/go_ultra`（监听 :8080）。`%~dp0` 是本脚本所在目录（`scripts\`），`%~dp0..` 即仓库根。

  ```bat
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
  ```

  说明：
  - 用 `cmd /k` 而非 `/B`：开发期需要看到各自的实时日志且能单独 `Ctrl+C`，所以各开一个可见窗口并保持打开（`/k`）。
  - `cd /d`：跨盘符切换目录（仓库在 `E:`，确保切换成功）。
  - 路径里有空格也安全：内层命令字符串用了嵌套引号。

- [ ] **手动验证**

  在仓库根执行：
  ```bat
  scripts\dev.bat
  ```
  预期：弹出两个新命令行窗口。
  - 标题为 `go_ultra-dev-api` 的窗口最终出现 Gin 启动日志，监听 `:8080`。
  - 标题为 `go_ultra-dev-web` 的窗口出现 Vite 日志，类似：
    ```
    VITE v5.x.x  ready in xxx ms
    ➜  Local:   http://localhost:5173/
    ```
  浏览器访问 `http://localhost:5173` 能打开页面、`http://localhost:8080/api/healthz` 返回 `{"status":"ok"}` 即通过。验证完关闭两个窗口。

- [ ] **commit**
  ```bat
  git add scripts/dev.bat
  git commit -m "feat(deploy): add scripts/dev.bat to run vite dev and go run in parallel"
  ```

---

### Task 7.3: 编写 scripts/build.bat（前端构建 + 后端编译）

**Files:**
- Create: `scripts/build.bat`
- Verify: 手动运行（无单元测试）

步骤：

- [ ] **创建 `scripts/build.bat`（完整内容，可直接粘贴）**

  先 `pnpm build` 产出 `web/dist`，再 `go build` 产出 `server/go_ultra.exe`。任一步失败即终止并返回非零退出码（用 `if errorlevel 1`）。

  ```bat
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
  ```

  说明：
  - 用 `call pnpm ...`：`pnpm` 是 `.cmd` 批处理，不加 `call` 会在第一条 pnpm 命令后直接退出本脚本。
  - `pnpm install --frozen-lockfile`：CI/可复现构建，锁定 `pnpm-lock.yaml`（spec §5 目录里有该文件）。
  - `go build -o go_ultra.exe ./cmd/go_ultra`：输出到 `server/go_ultra.exe`（spec §5 目录结构与 §3.2 进程名一致）。
  - `pushd`/`popd` 配对，保证无论成功失败都回到原目录。

- [ ] **手动验证**

  在仓库根执行：
  ```bat
  scripts\build.bat
  echo exit code = %errorlevel%
  ```
  预期：
  - 末尾打印 `[build] done.` 且 `exit code = 0`。
  - `web\dist\index.html` 存在。
  - `server\go_ultra.exe` 存在：
    ```bat
    dir server\go_ultra.exe
    ```
    应列出该文件。
  若任一步失败，脚本会打印对应 `FAILED:` 行且 `exit code` 非 0。

- [ ] **commit**
  ```bat
  git add scripts/build.bat
  git commit -m "feat(deploy): add scripts/build.bat to build web dist and go_ultra.exe"
  ```

---

### Task 7.4: 编写 start.bat（建 logs → 启后端 → 探活 healthz → 启 caddy → 启 cloudflared）

**Files:**
- Create: `start.bat`
- Verify: 手动运行（无单元测试）

**前置依赖：** 后端必须已实现 `GET /api/healthz` 返回 200 `{"status":"ok"}`（契约 http 层第 219 行），且已执行过 `scripts\build.bat` 生成 `server\go_ultra.exe`；`caddy`、`cloudflared` 在 `PATH` 中（Cloudflare Tunnel 一次性配置见 Task 7.6）。

步骤：

- [ ] **创建 `start.bat`（完整内容，可直接粘贴）**

  流程严格按要求：先 `mkdir logs` → 启动 `go_ultra.exe` → 轮询 `GET /api/healthz` 直到返回 200 才继续 → 启动 `caddy` → 启动 `cloudflared`。健康探测用 `curl`（Windows 10/11 自带 `curl.exe`）取 HTTP 状态码，循环最多 30 次（每次间隔约 1 秒）。

  ```bat
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
  ```

  说明：
  - `%~dp0` 已带结尾反斜杠，故拼成 `%ROOT%logs`、`%ROOT%server\go_ultra.exe`、`%ROOT%Caddyfile`（仓库根文件）。
  - `enabledelayedexpansion` + `!CODE!` / `!HEALTHY!`：在 `for` 循环体内读取被循环修改的变量必须用延迟展开 `!...!`，否则 `%CODE%` 在进入循环前就被固定。
  - `curl -s -o NUL -w "%%{http_code}"`：静默丢弃响应体，只把 HTTP 状态码写到 stdout；批处理里 `%` 要写成 `%%`。
  - 仅当探测到 `200` 才启动 caddy/cloudflared；30 次（约 30 秒）仍不健康则报错退出（`exit /b 1`），避免在后端没起来时反代到空端口。
  - 三个服务用 `/B`（后台、无独立窗口）启动，便于 `stop.bat` 统一 `taskkill`。
  - `cloudflared tunnel run go-ultra`：运行名为 `go-ultra` 的隧道（与 spec §8.3 一致；若已安装为 Windows 服务则该服务会自启，本行用于前台/手动启动场景）。

- [ ] **手动验证**

  确保已 `scripts\build.bat` 且 Tunnel 已配置（Task 7.6），然后在仓库根执行：
  ```bat
  start.bat
  ```
  预期输出包含：
  ```
  [start] preparing logs directory ...
  [start] launching go_ultra.exe (api :8080) ...
  [start] waiting for backend health at http://localhost:8080/api/healthz ...
  [start] not ready yet (attempt 1, http=000), retrying ...
  [start] backend healthy (attempt 2).
  [start] launching caddy (reverse proxy + tls) ...
  [start] launching cloudflared tunnel (go-ultra) ...
  [start] all services started.
  ```
  （前几次 `http=000` 是后端尚未监听端口时 curl 连接失败的正常现象；探到 `200` 后继续。）
  随后 `logs` 目录被创建，`https://localhost` 可访问，公网域名经 Tunnel 可达。
  若后端一直不健康，预期看到 `ERROR: backend did not become healthy after 30 attempts` 且退出码为 1（`echo %errorlevel%` 为 1）。

- [ ] **commit**
  ```bat
  git add start.bat
  git commit -m "feat(deploy): add start.bat that waits for /api/healthz before starting caddy and cloudflared"
  ```

---

### Task 7.5: 编写 stop.bat（taskkill 三个进程）

**Files:**
- Create: `stop.bat`
- Verify: 手动运行（无单元测试）

步骤：

- [ ] **创建 `stop.bat`（完整内容，可直接粘贴）**

  按映像名结束三个进程：`go_ultra.exe`、`caddy.exe`、`cloudflared.exe`。`/F` 强制结束，`/T` 连同子进程一并结束。逐个独立执行，单个不在运行不影响其它。

  ```bat
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
  ```

  说明：
  - `taskkill /IM <name> /F /T`：按映像名（image name）结束。若 `cloudflared` 是作为 Windows **服务**安装并运行的，`taskkill` 结束进程后服务可能被恢复管理器重启 —— 那种情况下应改用 `sc stop cloudflared` 或 `cloudflared service uninstall`；本脚本针对 `start.bat` 用 `/B` 手动拉起的前台进程场景。README（Task 7.6）会说明两种关闭方式。
  - `2>NUL`：屏蔽"进程不存在"的报错文本；进程不在运行时 `taskkill` 返回非零，由 `if errorlevel 1` 给出友好提示。

- [ ] **手动验证**

  先 `start.bat` 启动，再执行：
  ```bat
  stop.bat
  ```
  预期输出（三个均已停止时）：
  ```
  [stop] stopping go_ultra.exe ...
  [stop]   go_ultra.exe stopped.
  [stop] stopping caddy.exe ...
  [stop]   caddy.exe stopped.
  [stop] stopping cloudflared.exe ...
  [stop]   cloudflared.exe stopped.
  [stop] done.
  ```
  用任务管理器或下列命令确认三个进程已消失：
  ```bat
  tasklist /FI "IMAGENAME eq go_ultra.exe" /FI "IMAGENAME eq caddy.exe"
  ```
  预期输出为 `INFO: No tasks are running which match the specified criteria.`
  在没有任何服务运行时再次执行 `stop.bat`，预期对应行显示 `... not running.` 且脚本不报致命错误。

- [ ] **commit**
  ```bat
  git add stop.bat
  git commit -m "feat(deploy): add stop.bat to taskkill go_ultra, caddy and cloudflared"
  ```

---

### Task 7.6: README 运维补充章节（Tunnel 配置 / 管理员密码获取 / 手动备份 / reset 子命令说明）

**Files:**
- Create: `README.md`（若已存在则在文末追加"## 运维（Operations）"章节；本 Task 提供该章节完整内容）
- Verify: 手动核对命令可执行（`cloudflared --version` 等，无单元测试）

步骤：

- [ ] **在 `README.md` 文末追加以下"运维"章节（完整内容，可直接粘贴）**

  ````markdown
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
  ````

- [ ] **手动验证**

  README 是文档，验证标准是"其中给出的命令真实可执行、与脚本/子命令一致"：
  - 核对 cloudflared 可用：
    ```bat
    cloudflared --version
    ```
    预期打印版本号（形如 `cloudflared version 2024.x.x ...`）。
  - 核对 reset 子命令存在（依赖 Task 7.7 已完成）：
    ```bat
    cd server
    go_ultra.exe reset-admin-password
    ```
    预期打印新密码 banner，并生成/更新 `logs\admin_password.txt`。
  - 核对备份命令路径与脚本中数据库路径一致（与 Task 7.7 中 `dbPath` 默认值一致）。
  - 通读章节，确认无 spec §1.4 禁止的"具体游戏专有词汇"，全部使用中性术语。

- [ ] **commit**
  ```bat
  git add README.md
  git commit -m "docs(ops): add operations section (tunnel setup, admin password, backup, reset-admin-password)"
  ```

---

### Task 7.7: 在 main.go 实现 reset-admin-password 子命令

**Files:**
- Create/Modify: `server/cmd/go_ultra/main.go`
- Test: `server/cmd/go_ultra/main_test.go`

**契约对齐：** 密码生成与存储复用阶段 3 的 `AdminService`。本子命令需要"强制重新生成"，而 `EnsurePassword` 只在不存在时生成，因此新增一个 `AdminService.ResetPassword(ctx) (string, error)` 方法：调用阶段 3 已定义的包级函数 `GenerateAdminPassword()`（可读 16 位明文 + bcrypt）→ `SetSetting(adminPasswordHashKey, hash)` → 返回明文。命名与 `EnsurePassword`/`VerifyPassword` 同风格。

> 说明：`GenerateAdminPassword()` 与常量 `adminPasswordHashKey` **已在阶段 3（Task 6 `AdminService`）定义**，`EnsurePassword` 已改为调用它。本 Task **不重复定义** `GenerateAdminPassword`，只新增 `ResetPassword`，确保首启（`EnsurePassword`）与重置（`ResetPassword`）走同一密码生成实现、写同一 setting key。

步骤：

- [ ] **先写失败测试 `server/cmd/go_ultra/main_test.go`（完整内容，可直接粘贴）**

  测试不真正启动服务，只验证参数分发函数 `dispatch` 对 `reset-admin-password` 返回正确动作、对未知参数返回用法+退出码 2、无参数返回 serve。把可测逻辑抽成 `dispatch(args []string) (action string, code int)`。

  ```go
  package main

  import "testing"

  func TestDispatch(t *testing.T) {
  	tests := []struct {
  		name       string
  		args       []string
  		wantAction string
  		wantCode   int
  	}{
  		{"no args runs server", []string{}, "serve", 0},
  		{"reset subcommand", []string{"reset-admin-password"}, "reset-admin-password", 0},
  		{"unknown subcommand", []string{"frobnicate"}, "usage", 2},
  		{"too many args", []string{"reset-admin-password", "extra"}, "usage", 2},
  	}
  	for _, tt := range tests {
  		t.Run(tt.name, func(t *testing.T) {
  			action, code := dispatch(tt.args)
  			if action != tt.wantAction {
  				t.Errorf("dispatch(%v) action = %q, want %q", tt.args, action, tt.wantAction)
  			}
  			if code != tt.wantCode {
  				t.Errorf("dispatch(%v) code = %d, want %d", tt.args, code, tt.wantCode)
  			}
  		})
  	}
  }
  ```

- [ ] **运行确认失败**
  ```bat
  cd server
  go test ./cmd/go_ultra/
  ```
  预期失败（`dispatch` 未定义）：
  ```
  # go_ultra/cmd/go_ultra [go_ultra/cmd/go_ultra.test]
  .\main_test.go:18:20: undefined: dispatch
  FAIL    go_ultra/cmd/go_ultra [build failed]
  ```

- [ ] **在 `server/internal/service/admin.go` 追加 `ResetPassword`（完整内容，可直接粘贴；`GenerateAdminPassword` 与 `adminPasswordHashKey` 已在阶段 3 定义，此处直接调用）**

  ```go
  // ResetPassword 强制重新生成管理员密码（覆盖已有 admin_password_hash），返回新明文。
  func (s *AdminService) ResetPassword(ctx context.Context) (string, error) {
  	plaintext, hash, err := GenerateAdminPassword()
  	if err != nil {
  		return "", domain.ErrInternal.WithCause(err)
  	}
  	if err := s.q.SetSetting(ctx, sqlc.SetSettingParams{
  		Key:   adminPasswordHashKey,
  		Value: hash,
  	}); err != nil {
  		return "", domain.ErrInternal.WithCause(err)
  	}
  	return plaintext, nil
  }
  ```

  > 注：`adminPasswordHashKey`（阶段 3 定义的常量 `"admin_password_hash"`）与 `GenerateAdminPassword`（阶段 3 包级函数）均已存在，`ResetPassword` 直接引用，无需新增 import（`context`/`sqlc`/`domain` 已在 admin.go 中）。

- [ ] **修改 `server/cmd/go_ultra/main.go`，增量加入子命令分发（不重写阶段 4 的装配逻辑）**

  > **合并约定（务必遵守）**：`main.go` 由阶段 4（Task 13）创建并拥有，已含 `buildRouter(cfg config.Config) (*gin.Engine, func(), error)` 与 `main()`（读取 `config.Load()` → `buildRouter` → `r.Run(cfg.Addr)`）。本 Task **只做增量修改**：抽出一个纯函数 `dispatch`，让 `main()` 先分发命令，再走原有 serve 路径或新的 reset 路径。**不要**新建 `serve.go`、**不要**自造 `startHTTP`/`runServe`/`dbPath` 常量或 `select{}` 占位 —— serve 路径直接复用阶段 4 的 `buildRouter` + `r.Run`。

  阶段 4（Task 13）实际写入的 `main()` 原形如下（逐字对照，Edit 的 old_string 必须用这一版）：

  ```go
  func main() {
  	gin.SetMode(gin.ReleaseMode)
  	cfg := config.Load()
  	logger := newLogger()

  	router, cleanup, err := buildRouter(cfg)
  	if err != nil {
  		logger.Fatal().Err(err).Msg("failed to build router")
  	}
  	defer cleanup()

  	logger.Info().Str("addr", cfg.Addr).Msg("go_ultra listening")
  	if err := router.Run(cfg.Addr); err != nil {
  		logger.Fatal().Err(err).Msg("server stopped")
  	}
  }
  ```

  用 Edit 把上面的整个 `func main() { ... }` 替换为下面的版本（新增 `dispatch`/`runReset`/`emitAdminPassword`；serve 分支与阶段 4 行为逐字一致——保留 `gin.SetMode(gin.ReleaseMode)`、`newLogger()`、`logger.Fatal()`/`logger.Info()`）：

  ```go
  // adminPasswordFile 是管理员明文密码落盘路径（权限说明见 README 运维章节）。
  const adminPasswordFile = "logs/admin_password.txt"

  // dispatch 解析命令行参数，返回要执行的动作与建议退出码。纯函数，便于单元测试。
  func dispatch(args []string) (action string, code int) {
  	if len(args) == 0 {
  		return "serve", 0
  	}
  	if len(args) == 1 && args[0] == "reset-admin-password" {
  		return "reset-admin-password", 0
  	}
  	return "usage", 2
  }

  func main() {
  	action, code := dispatch(os.Args[1:])
  	switch action {
  	case "serve":
  		gin.SetMode(gin.ReleaseMode)
  		cfg := config.Load()
  		logger := newLogger()

  		router, cleanup, err := buildRouter(cfg)
  		if err != nil {
  			logger.Fatal().Err(err).Msg("failed to build router")
  		}
  		defer cleanup()

  		logger.Info().Str("addr", cfg.Addr).Msg("go_ultra listening")
  		if err := router.Run(cfg.Addr); err != nil {
  			logger.Fatal().Err(err).Msg("server stopped")
  		}
  	case "reset-admin-password":
  		if err := runReset(); err != nil {
  			fmt.Fprintln(os.Stderr, "fatal:", err)
  			os.Exit(1)
  		}
  	default: // "usage"
  		fmt.Fprintln(os.Stderr, "usage: go_ultra [reset-admin-password]")
  		os.Exit(code)
  	}
  }

  // runReset 重置管理员密码并把新明文输出到 stdout 与 logs/admin_password.txt。
  // 复用阶段 4 的 config 与 db 装配，不另起一套数据库路径常量。
  func runReset() error {
  	cfg := config.Load()
  	database, err := db.New(cfg.DBPath)
  	if err != nil {
  		return err
  	}
  	defer database.Close()

  	admin := service.NewAdminService(sqlc.New(database), database)
  	plaintext, err := admin.ResetPassword(context.Background())
  	if err != nil {
  		return err
  	}
  	return emitAdminPassword("reset", plaintext)
  }

  // emitAdminPassword 把明文密码打印到 stdout 并写入 logs/admin_password.txt（0600）。
  func emitAdminPassword(kind, plaintext string) error {
  	banner := "============================================================"
  	fmt.Println(banner)
  	fmt.Printf("ADMIN PASSWORD (%s, shown only once): %s\n", kind, plaintext)
  	fmt.Printf("also written to %s\n", adminPasswordFile)
  	fmt.Println(banner)

  	if err := os.MkdirAll(filepath.Dir(adminPasswordFile), 0o755); err != nil {
  		return err
  	}
  	return os.WriteFile(adminPasswordFile, []byte(plaintext+"\n"), 0o600)
  }
  ```

  并确保 `main.go` 顶部 import 包含 `"context"`、`"fmt"`、`"os"`、`"path/filepath"`、`"go_ultra/internal/config"`、`"go_ultra/internal/db"`、`"go_ultra/internal/db/sqlc"`、`"go_ultra/internal/service"`、`"github.com/gin-gonic/gin"`（阶段 4 已引入大部分，缺 `fmt`/`path/filepath`/`sqlc` 则补上）。

  > 说明：serve 分支与阶段 4 的原 `main` 行为逐字一致（`gin.SetMode` → `config.Load` → `newLogger` → `buildRouter` → `logger.Info` → `router.Run`，`buildRouter` 内部已调用 `EnsurePassword` 并在首次生成时打印密码）。reset 分支不启动 HTTP，仅重置密码后退出。无 `startHTTP`/`select{}` 占位，整包可直接编译。


- [ ] **运行确认通过**
  ```bat
  cd server
  go test ./cmd/go_ultra/
  ```
  预期输出：
  ```
  ok      go_ultra/cmd/go_ultra   0.0xxs
  ```
  并确认整包可编译：
  ```bat
  go build ./...
  ```
  预期无输出（退出码 0）。

- [ ] **手动验证子命令端到端**
  ```bat
  cd server
  go run ./cmd/go_ultra reset-admin-password
  ```
  预期 stdout 打印密码 banner，且：
  ```bat
  type logs\admin_password.txt
  ```
  显示同一明文密码。再验证未知参数：
  ```bat
  go run ./cmd/go_ultra frobnicate
  echo %errorlevel%
  ```
  预期打印 `usage: go_ultra [reset-admin-password]` 且退出码为 `2`。

- [ ] **commit**
  ```bat
  git add server/cmd/go_ultra/main.go server/cmd/go_ultra/main_test.go server/internal/service/admin.go
  git commit -m "feat(cmd): add reset-admin-password subcommand and reusable admin password generation"
  ```
