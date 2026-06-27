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

func TestLoad_AllowedOriginsDefault(t *testing.T) {
	t.Setenv("GO_ULTRA_DB", "")
	cfg := Load()
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("AllowedOrigins = %v, want [http://localhost:5173]", cfg.AllowedOrigins)
	}
}

func TestLoad_AllowedOriginsOverride(t *testing.T) {
	t.Setenv("GO_ULTRA_ALLOWED_ORIGINS", "https://go-ultra.example.com, http://localhost:5173")
	cfg := Load()
	if len(cfg.AllowedOrigins) != 2 ||
		cfg.AllowedOrigins[0] != "https://go-ultra.example.com" ||
		cfg.AllowedOrigins[1] != "http://localhost:5173" {
		t.Fatalf("AllowedOrigins = %v, want [https://go-ultra.example.com http://localhost:5173]", cfg.AllowedOrigins)
	}
}
