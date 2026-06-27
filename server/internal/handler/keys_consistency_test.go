package handler

import (
	"net/http/httptest"
	"testing"

	"go_ultra/internal/middleware"

	"github.com/gin-gonic/gin"
)

// TestContextKeyConsistency proves handler's private ctx keys and middleware's
// exported Ctx keys reference the SAME gin.Context slots. Middleware writes using
// its constants; handlers read using theirs. If anyone changes one set, this fails.
func TestContextKeyConsistency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Write via middleware's exported keys...
	c.Set(middleware.CtxLogger, "L")
	c.Set(middleware.CtxRequestID, "R")
	c.Set(middleware.CtxPlayerID, int64(7))

	// ...read via handler's private keys. Must round-trip.
	if v, ok := c.Get(ctxLogger); !ok || v.(string) != "L" {
		t.Fatalf("ctxLogger key mismatch: handler %q vs middleware %q", ctxLogger, middleware.CtxLogger)
	}
	if v, ok := c.Get(ctxRequestID); !ok || v.(string) != "R" {
		t.Fatalf("ctxRequestID key mismatch: handler %q vs middleware %q", ctxRequestID, middleware.CtxRequestID)
	}
	if v, ok := c.Get(ctxPlayerID); !ok || v.(int64) != 7 {
		t.Fatalf("ctxPlayerID key mismatch: handler %q vs middleware %q", ctxPlayerID, middleware.CtxPlayerID)
	}
}
