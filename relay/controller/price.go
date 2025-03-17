package controller

import (
	"github.com/labring/aiproxy/model"
)

func GetModelPrice(modelConfig *model.ModelConfig) (float64, float64, float64, float64, bool) {
	return modelConfig.InputPrice, modelConfig.OutputPrice, modelConfig.CachedPrice, modelConfig.CacheCreationPrice, true
}

func GetImageSizePrice(modelConfig *model.ModelConfig, size string) (float64, bool) {
	if len(modelConfig.ImagePrices) == 0 {
		return 0, true
	}
	price, ok := modelConfig.ImagePrices[size]
	return price, ok
}
