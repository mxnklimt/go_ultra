package main

import (
	"context"
	"fmt"
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
		Player:         playerSvc,
		Match:          matchSvc,
		Leaderboard:    leaderboardSvc,
		Admin:          adminSvc,
		Logger:         logger,
		AllowedOrigins: cfg.AllowedOrigins,
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
		// 停止运行中后端以避免 DB 写冲突（与 README 运维章节指引一致）。
		os.Exit(runReset())

	case "usage":
		os.Stderr.WriteString("usage: go_ultra [reset-admin-password]\n")
	}
	os.Exit(code)
}

// runReset 独立完成 reset 全程（打开 DB、构造 service、写密码、输出 banner），
// 不复用 buildRouter（reset 不需要 router / logger 装配）。
// 返回退出码：0 成功，1 失败。
func runReset() int {
	cfg := config.Load()

	sqlDB, err := db.New(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		return 1
	}
	defer sqlDB.Close()

	q := sqlc.New(sqlDB)
	adminSvc := service.NewAdminService(q, sqlDB)

	plaintext, err := adminSvc.ResetPassword(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "reset password: %v\n", err)
		return 1
	}

	emitAdminPassword(plaintext)
	return 0
}

// emitAdminPassword 把重置后的明文密码打印到 stdout 并写 logs/admin_password.txt。
// 与 EnsurePassword 的"首启打印"共享同一输出格式，保证运维体验一致。
func emitAdminPassword(plaintext string) {
	os.Stdout.WriteString("\n===================================================\n")
	os.Stdout.WriteString("ADMIN PASSWORD (reset, shown only once):\n")
	os.Stdout.WriteString(plaintext + "\n")
	os.Stdout.WriteString("also written to " + adminPasswordFile + "\n")
	os.Stdout.WriteString("===================================================\n\n")

	if err := writeAdminPassword(plaintext); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", adminPasswordFile, err)
	}
}
