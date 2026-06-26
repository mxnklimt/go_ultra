package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go_ultra/internal/config"
)

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

func TestBuildRouter_Healthz(t *testing.T) {
	t.Cleanup(func() { os.RemoveAll("logs") })
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
