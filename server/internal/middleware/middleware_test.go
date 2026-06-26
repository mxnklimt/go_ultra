package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequestID_SetsHeaderAndContext(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	var gotFromCtx string
	r.GET("/", func(c *gin.Context) {
		v, _ := c.Get(CtxRequestID)
		gotFromCtx, _ = v.(string)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	hdr := w.Header().Get("X-Request-ID")
	if hdr == "" {
		t.Fatalf("X-Request-ID header is empty")
	}
	if gotFromCtx == "" {
		t.Fatalf("request id not stored in context")
	}
	if hdr != gotFromCtx {
		t.Fatalf("header %q != context %q", hdr, gotFromCtx)
	}
}

func TestLogger_InjectsLoggerIntoContext(t *testing.T) {
	r := gin.New()
	base := zerolog.Nop()
	r.Use(RequestID())
	r.Use(Logger(base))
	found := false
	r.GET("/", func(c *gin.Context) {
		if _, ok := c.Get(CtxLogger); ok {
			found = true
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if !found {
		t.Fatalf("logger not injected into context")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestRecover_PanicBecomes500JSON(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger(zerolog.Nop()))
	r.Use(Recover())
	r.GET("/boom", func(c *gin.Context) {
		panic("kaboom")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body %q: %v", w.Body.String(), err)
	}
	if body.Error.Code != "INTERNAL" {
		t.Fatalf("code = %q, want INTERNAL", body.Error.Code)
	}
}
