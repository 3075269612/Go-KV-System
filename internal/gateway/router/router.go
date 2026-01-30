package router

import (
	"Go-AI-KV-System/internal/gateway/handler"

	"github.com/gin-gonic/gin"
)

// NewRouter 初始化 Gin 引擎并注册路由
func NewRouter(healthHandler *handler.HealthHandler) *gin.Engine {
	r := gin.Default()

	// 基础健康检查
	r.GET("/ping", healthHandler.Ping)
	
	return r
}