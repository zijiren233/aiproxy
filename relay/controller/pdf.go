package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
)

func GetPdfRequestPrice(c *gin.Context, mc *model.ModelConfig) (*model.Price, error) {
	return &mc.Price, nil
}

func GetPdfRequestUsage(c *gin.Context, mc *model.ModelConfig) (*model.Usage, error) {
	return &model.Usage{
		InputTokens: 1,
	}, nil
}
