package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// 与 handler 包共用的 gin.Context key（值必须一致）。
const (
	CtxLogger    = "logger"    // zerolog.Logger
	CtxRequestID = "requestID" // string
	CtxPlayerID  = "playerID"  // int64
)

// RequestID 为每个请求生成一个 UUID，存入 context 并写入响应头 X-Request-ID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		c.Set(CtxRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger 注入一个带 request_id 字段的请求级 logger，并在请求结束后输出一行访问日志。
func Logger(base zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID, _ := c.Get(CtxRequestID)
		ridStr, _ := reqID.(string)

		reqLogger := base.With().Str("request_id", ridStr).Logger()
		c.Set(CtxLogger, reqLogger)

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		var playerID int64
		if v, ok := c.Get(CtxPlayerID); ok {
			if pid, ok := v.(int64); ok {
				playerID = pid
			}
		}

		ev := reqLogger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Int64("latency_ms", latency.Milliseconds())
		if playerID != 0 {
			ev = ev.Int64("player_id", playerID)
		}
		if len(c.Errors) > 0 {
			ev = ev.Str("error", c.Errors.String())
		}
		ev.Msg("http request")
	}
}

// Recover 捕获 handler 中的 panic，返回统一的 500 JSON，并记录堆栈信息。
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if v, ok := c.Get(CtxLogger); ok {
					if lg, ok := v.(zerolog.Logger); ok {
						lg.Error().Interface("panic", rec).Bytes("stack", debug.Stack()).Msg("recovered from panic")
					}
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "INTERNAL",
						"message": "服务器内部错误",
					},
				})
			}
		}()
		c.Next()
	}
}
