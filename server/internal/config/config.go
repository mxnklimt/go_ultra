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
