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
