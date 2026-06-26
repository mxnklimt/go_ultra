package handler

import (
	"net/http"

	"go_ultra/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// gin.Context 中存放共享值的 key 常量。
const (
	ctxLogger    = "logger"    // zerolog.Logger
	ctxRequestID = "requestID" // string
	ctxPlayerID  = "playerID"  // int64
)

// errorEnvelope 是统一错误响应的 JSON 形状：{"error":{"code","message"}}。
type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// loggerFrom 从 gin.Context 取出请求级 logger；不存在则返回 Nop。
func loggerFrom(c *gin.Context) zerolog.Logger {
	if v, ok := c.Get(ctxLogger); ok {
		if lg, ok := v.(zerolog.Logger); ok {
			return lg
		}
	}
	return zerolog.Nop()
}

// respondError 把任意 error 转换为统一错误响应并终止请求链。
// *domain.Error 直接取 Status/Code/Message；其它 error 一律 500 INTERNAL，
// 并把原始 error 作为 Cause 记入日志。
func respondError(c *gin.Context, err error) {
	var de *domain.Error
	if e, ok := err.(*domain.Error); ok && e != nil {
		de = e
	} else {
		de = domain.ErrInternal.WithCause(err)
	}

	if de.Status >= http.StatusInternalServerError {
		lg := loggerFrom(c)
		ev := lg.Error().
			Str("code", de.Code).
			Str("message", de.Message)
		if de.Cause != nil {
			ev = ev.Err(de.Cause)
		}
		ev.Msg("request failed")
	}

	c.AbortWithStatusJSON(de.Status, errorEnvelope{
		Error: errorPayload{Code: de.Code, Message: de.Message},
	})
}
