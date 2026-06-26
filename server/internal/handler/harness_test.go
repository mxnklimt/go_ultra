package handler

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go_ultra/internal/db"
	"go_ultra/internal/db/sqlc"
	"go_ultra/internal/service"

	"github.com/rs/zerolog"
)

// testServer 持有一个真实装配的 router 与底层依赖，供 HTTP 测试使用。
type testServer struct {
	t      *testing.T
	router http.Handler
	deps   Deps
}

// newTestServer 用临时 sqlite 文件库 + 真实 service 构造一个 router。
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("db.New failed: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	q := sqlc.New(sqlDB)
	deps := Deps{
		Player:      service.NewPlayerService(q, sqlDB),
		Match:       service.NewMatchService(q, sqlDB),
		Leaderboard: service.NewLeaderboardService(q, sqlDB),
		Admin:       service.NewAdminService(q, sqlDB),
		Logger:      zerolog.Nop(),
	}
	return &testServer{
		t:      t,
		router: NewRouter(deps),
		deps:   deps,
	}
}

// do 发起一个请求并返回 recorder。body 为已序列化好的 JSON 字符串（可空）。
func (ts *testServer) do(method, path, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	ts.t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	for _, ck := range cookies {
		r.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, r)
	return w
}

// login 以给定用户名登录（隐式注册），返回玩家会话 cookie。
func (ts *testServer) login(username string) *http.Cookie {
	ts.t.Helper()
	w := ts.do(http.MethodPost, "/api/login", `{"username":"`+username+`"}`)
	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		ts.t.Fatalf("login(%q) status = %d, body=%s", username, w.Code, w.Body.String())
	}
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "go_ultra_session" {
			return ck
		}
	}
	ts.t.Fatalf("login(%q) did not set go_ultra_session cookie", username)
	return nil
}

// adminLogin 取出首启生成的管理员明文密码并登录，返回 admin cookie。
func (ts *testServer) adminLogin(plaintext string) *http.Cookie {
	ts.t.Helper()
	w := ts.do(http.MethodPost, "/api/admin/login", `{"password":"`+plaintext+`"}`)
	if w.Code != http.StatusOK {
		ts.t.Fatalf("admin login status = %d, body=%s", w.Code, w.Body.String())
	}
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "go_ultra_admin" {
			return ck
		}
	}
	ts.t.Fatalf("admin login did not set go_ultra_admin cookie")
	return nil
}
