package router

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/docs"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetSwaggerRouter(router *gin.Engine) {
	docs.SwaggerInfo.BasePath = "/api"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
