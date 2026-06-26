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
