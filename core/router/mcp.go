package router

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/controller"
	"github.com/labring/aiproxy/core/middleware"
)

func SetMCPRouter(router *gin.Engine) {
	mcpRoute := router.Group("/mcp", middleware.MCPAuth)

	mcpRoute.GET("/public/:id/sse", controller.MCPSseProxy)
	mcpRoute.POST("/message", controller.MCPMessage)
}
