package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type fakePlayerChecker struct {
	playerID int64
	ok       bool
	err      error
	gotToken string
}

func (f *fakePlayerChecker) GetSession(ctx context.Context, token string) (int64, bool, error) {
	f.gotToken = token
	return f.playerID, f.ok, f.err
}

type fakeAdminChecker struct {
	ok        bool
	expiresAt time.Time
	err       error
	gotToken  string
}

func (f *fakeAdminChecker) CheckAdminSession(ctx context.Context, token string) (bool, time.Time, error) {
	f.gotToken = token
	return f.ok, f.expiresAt, f.err
}

func decodeErrCode(t *testing.T, body []byte) string {
	t.Helper()
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &b); err != nil {
		t.Fatalf("invalid json %q: %v", string(body), err)
	}
	return b.Error.Code
}

func TestPlayerAuth_NoCookie_401(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(&fakePlayerChecker{}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if code := decodeErrCode(t, w.Body.Bytes()); code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %q, want NOT_AUTHENTICATED", code)
	}
}

func TestPlayerAuth_InvalidSession_401(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(&fakePlayerChecker{ok: false}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_session", Value: "expired-or-unknown"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if code := decodeErrCode(t, w.Body.Bytes()); code != "NOT_AUTHENTICATED" {
		t.Fatalf("code = %q, want NOT_AUTHENTICATED", code)
	}
}

func TestPlayerAuth_Valid_InjectsPlayerID(t *testing.T) {
	checker := &fakePlayerChecker{playerID: 42, ok: true}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(checker))
	var got int64
	r.GET("/", func(c *gin.Context) {
		if v, ok := c.Get(CtxPlayerID); ok {
			got, _ = v.(int64)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_session", Value: "good-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got != 42 {
		t.Fatalf("playerID = %d, want 42", got)
	}
	if checker.gotToken != "good-token" {
		t.Fatalf("checker received token %q, want good-token", checker.gotToken)
	}
}

func TestPlayerAuth_CheckerError_500(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(PlayerAuth(&fakePlayerChecker{err: context.DeadlineExceeded}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_session", Value: "x"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestAdminAuth_NoCookie_403(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(AdminAuth(&fakeAdminChecker{}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if code := decodeErrCode(t, w.Body.Bytes()); code != "ADMIN_REQUIRED" {
		t.Fatalf("code = %q, want ADMIN_REQUIRED", code)
	}
}

func TestAdminAuth_InvalidSession_403(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(AdminAuth(&fakeAdminChecker{ok: false}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_admin", Value: "nope"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestAdminAuth_Valid_Passes(t *testing.T) {
	checker := &fakeAdminChecker{ok: true, expiresAt: time.Now().Add(time.Minute)}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxLogger, zerolog.Nop()); c.Next() })
	r.Use(AdminAuth(checker))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "go_ultra_admin", Value: "admin-token"})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if checker.gotToken != "admin-token" {
		t.Fatalf("checker received token %q, want admin-token", checker.gotToken)
	}
}

// 确保 domain 错误码字符串与中间件使用一致（防御 typo）。
func TestAuth_DomainCodes(t *testing.T) {
	if domain.ErrNotAuthenticated.Code != "NOT_AUTHENTICATED" {
		t.Fatalf("unexpected NOT_AUTHENTICATED code: %q", domain.ErrNotAuthenticated.Code)
	}
	if domain.ErrAdminRequired.Code != "ADMIN_REQUIRED" {
		t.Fatalf("unexpected ADMIN_REQUIRED code: %q", domain.ErrAdminRequired.Code)
	}
}
