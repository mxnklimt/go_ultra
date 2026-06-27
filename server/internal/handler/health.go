package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleHealthz 返回服务存活探针响应。
func handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
