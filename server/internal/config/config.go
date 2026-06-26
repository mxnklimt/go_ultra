package config

import "os"

// Config 持有运行期配置。
type Config struct {
	DBPath         string   // SQLite 文件路径
	Addr           string   // HTTP 监听地址
	AllowedOrigins []string // CSRF Origin 头白名单（精确匹配）
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
		// 开发期 Vite dev server 源；生产部署时追加 Cloudflare Tunnel 域名
		// （如 "https://go-ultra.example.com"）。
		AllowedOrigins: []string{"http://localhost:5173"},
	}
}
