package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
)

func GetPdfRequestPrice(_ *gin.Context, mc model.ModelConfig) (model.Price, error) {
	return mc.Price, nil
}

func GetPdfRequestUsage(_ *gin.Context, _ model.ModelConfig) (model.Usage, error) {
	return model.Usage{}, nil
}
