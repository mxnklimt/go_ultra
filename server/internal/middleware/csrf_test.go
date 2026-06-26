package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newOriginTestEngine 构造一个最小 gin engine：根中间件挂 OriginCheck，
// 注册一个对所有方法都返回 200 的探针路由，便于断言"放行 vs 拒绝"。
func newOriginTestEngine(allowed []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OriginCheck(allowed))
	handler := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	r.GET("/probe", handler)
	r.POST("/probe", handler)
	r.DELETE("/probe", handler)
	return r
}

func TestOriginCheck(t *testing.T) {
	const allowedOrigin = "https://go-ultra.example.com"
	allowed := []string{allowedOrigin, "http://localhost:5173"}

	tests := []struct {
		name       string
		method     string
		setOrigin  bool
		origin     string
		wantStatus int
	}{
		{
			name:       "safe GET passes without origin",
			method:     http.MethodGet,
			setOrigin:  false,
			wantStatus: http.StatusOK,
		},
		{
			name:       "safe GET passes with foreign origin",
			method:     http.MethodGet,
			setOrigin:  true,
			origin:     "https://evil.example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with allowed origin passes",
			method:     http.MethodPost,
			setOrigin:  true,
			origin:     allowedOrigin,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with dev origin passes",
			method:     http.MethodPost,
			setOrigin:  true,
			origin:     "http://localhost:5173",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with foreign origin rejected",
			method:     http.MethodPost,
			setOrigin:  true,
			origin:     "https://evil.example.com",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST with missing origin rejected",
			method:     http.MethodPost,
			setOrigin:  false,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DELETE with foreign origin rejected",
			method:     http.MethodDelete,
			setOrigin:  true,
			origin:     "https://evil.example.com",
			wantStatus: http.StatusBadRequest,
		},
	}

	r := newOriginTestEngine(allowed)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/probe", nil)
			if tc.setOrigin {
				req.Header.Set("Origin", tc.origin)
			}
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
			// 被拒时应返回统一错误格式，code 为 INVALID_PARAM。
			if tc.wantStatus == http.StatusBadRequest {
				if body := w.Body.String(); body != `{"error":{"code":"INVALID_PARAM","message":"参数无效"}}` {
					t.Fatalf("error body = %q, want INVALID_PARAM envelope", body)
				}
			}
		})
	}
}
