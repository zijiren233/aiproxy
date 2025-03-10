package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common"
	"github.com/labring/aiproxy/middleware"
)

type StatusData struct {
	StartTime int64 `json:"startTime"`
}

func GetStatus(c *gin.Context) {
	middleware.SuccessResponse(c, &StatusData{
		StartTime: common.StartTime,
	})
}
