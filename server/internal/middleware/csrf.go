package middleware

import (
	"net/http"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
)

// safeMethods 是无副作用、无需 CSRF 防护的 HTTP 方法。
var safeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// OriginCheck 对非安全方法（POST/PUT/PATCH/DELETE）校验 Origin 头。
//
// 策略（fail-closed）：
//   - 安全方法（GET/HEAD/OPTIONS）直接放行，不读 Origin。
//   - 非安全方法：Origin 头缺失视为可疑（浏览器对跨站写请求必带 Origin），拒绝。
//   - 非安全方法：Origin 须精确等于 allowedOrigins 中的某一项，否则拒绝。
//
// 拒绝时统一返回 domain.ErrInvalidParam（400 / INVALID_PARAM）。
func OriginCheck(allowedOrigins []string) gin.HandlerFunc {
	// 预构建集合，O(1) 查找，避免每请求线性扫描。
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(c *gin.Context) {
		if safeMethods[c.Request.Method] {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			abort(c, domain.ErrInvalidParam)
			return
		}
		if !allowed[origin] {
			abort(c, domain.ErrInvalidParam)
			return
		}

		c.Next()
	}
}
