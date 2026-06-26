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
