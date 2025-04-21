package router

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/controller"
	"github.com/labring/aiproxy/core/middleware"
)

func SetMCPRouter(router *gin.Engine) {
	mcpRoute := router.Group("/mcp", middleware.MCPAuth)

	mcpRoute.GET("/public/:id/sse", controller.PublicMCPSseServer)
	mcpRoute.POST("/public/message", controller.PublicMCPMessage)

	mcpRoute.GET("/group/:id/sse", controller.GroupMCPSseServer)
	mcpRoute.POST("/group/message", controller.GroupMCPMessage)
}
