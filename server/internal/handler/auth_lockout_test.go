package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go_ultra/internal/domain"
)

// fakeAdminAuth 实现 authAdminService，用于在不依赖 DB 的情况下测试退避分支。
type fakeAdminAuth struct {
	locked       bool
	pwOK         bool
	recordCalled int
	resetCalled  int
}

func (f *fakeAdminAuth) VerifyPassword(_ context.Context, _ string) (bool, error) { return f.pwOK, nil }
func (f *fakeAdminAuth) CreateAdminSession(_ context.Context) (string, time.Time, error) {
	return "tok", time.Now().Add(30 * time.Minute), nil
}
func (f *fakeAdminAuth) CheckAdminSession(_ context.Context, _ string) (bool, time.Time, error) {
	return true, time.Now().Add(30 * time.Minute), nil
}
func (f *fakeAdminAuth) DeleteAdminSession(_ context.Context, _ string) error { return nil }
func (f *fakeAdminAuth) CheckLockout() error {
	if f.locked {
		return domain.ErrRateLimited
	}
	return nil
}
func (f *fakeAdminAuth) RecordLoginFailure() { f.recordCalled++ }
func (f *fakeAdminAuth) ResetLoginFailures() { f.resetCalled++ }

// doAdminLogin 构造一个仅挂载 admin login 路由的引擎并发起请求。
// handleAdminLogin 是 *authHandler 上的方法（通过 h.admin 取服务），故这里
// 用 fake 构造一个 authHandler 再注册其方法，而非调用不存在的自由函数。
func doAdminLogin(t *testing.T, svc authAdminService, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &authHandler{admin: svc}
	r.POST("/api/admin/login", h.handleAdminLogin)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeErrCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return resp.Error.Code
}

func TestHandleAdminLogin_LockedReturns429(t *testing.T) {
	svc := &fakeAdminAuth{locked: true}
	w := doAdminLogin(t, svc, `{"password":"whatever"}`)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
	if code := decodeErrCode(t, w); code != "RATE_LIMITED" {
		t.Fatalf("error.code = %q, want RATE_LIMITED", code)
	}
	if svc.recordCalled != 0 {
		t.Errorf("RecordLoginFailure called %d times while locked, want 0", svc.recordCalled)
	}
}

func TestHandleAdminLogin_WrongPasswordRecordsFailure(t *testing.T) {
	svc := &fakeAdminAuth{locked: false, pwOK: false}
	w := doAdminLogin(t, svc, `{"password":"wrong"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if code := decodeErrCode(t, w); code != "INVALID_PARAM" {
		t.Fatalf("error.code = %q, want INVALID_PARAM", code)
	}
	if svc.recordCalled != 1 {
		t.Errorf("RecordLoginFailure called %d times, want 1", svc.recordCalled)
	}
	if svc.resetCalled != 0 {
		t.Errorf("ResetLoginFailures called %d times on failure, want 0", svc.resetCalled)
	}
}

func TestHandleAdminLogin_SuccessResetsFailures(t *testing.T) {
	svc := &fakeAdminAuth{locked: false, pwOK: true}
	w := doAdminLogin(t, svc, `{"password":"correct"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if svc.resetCalled != 1 {
		t.Errorf("ResetLoginFailures called %d times, want 1", svc.resetCalled)
	}
	if svc.recordCalled != 0 {
		t.Errorf("RecordLoginFailure called %d times on success, want 0", svc.recordCalled)
	}
}
