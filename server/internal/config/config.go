package config

import (
	"os"
	"strings"
)

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
	// 开发期 Vite dev server 源；生产部署时通过 GO_ULTRA_ALLOWED_ORIGINS
	// 追加 Cloudflare Tunnel 域名（如 "https://go-ultra.example.com"），
	// 逗号分隔，未设置时回退到开发默认值。
	origins := []string{"http://localhost:5173"}
	if v := os.Getenv("GO_ULTRA_ALLOWED_ORIGINS"); v != "" {
		parts := strings.Split(v, ",")
		origins = origins[:0]
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				origins = append(origins, s)
			}
		}
	}
	return Config{
		DBPath:         dbPath,
		Addr:           ":8080",
		AllowedOrigins: origins,
	}
}
