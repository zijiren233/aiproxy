package controller

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func getVideoOutputPrice(
	modelConfig model.ModelConfig,
	size string,
	duration int64,
) (float64, bool) {
	if len(modelConfig.VideoPrices) == 0 {
		if modelConfig.Price.VideoPerSecondPrice > 0 {
			return float64(modelConfig.Price.VideoPerSecondPrice), true
		}
		return float64(modelConfig.Price.OutputPrice), true
	}

	priceItems, ok := modelConfig.VideoPrices[size]
	if !ok {
		return 0, false
	}

	for _, item := range priceItems {
		if duration >= item.SecondMin && (item.SecondMax == 0 || duration <= item.SecondMax) {
			return item.Price, true
		}
	}

	return 0, false
}

func GetVideoGenerationJobRequestPrice(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	videoGenerationJobRequest, err := utils.UnmarshalVideoGenerationJobRequest(c.Request)
	if err != nil {
		return model.Price{}, err
	}
	if videoGenerationJobRequest.NSeconds == 0 {
		return model.Price{}, errors.New("n_seconds is required")
	}
	if videoGenerationJobRequest.NVariants == 0 {
		return model.Price{}, errors.New("n_variants is required")
	}
	if videoGenerationJobRequest.Width == 0 {
		return model.Price{}, errors.New("width is required")
	}
	if videoGenerationJobRequest.Height == 0 {
		return model.Price{}, errors.New("height is required")
	}

	price, ok := getVideoOutputPrice(
		mc,
		fmt.Sprintf("%dx%d", videoGenerationJobRequest.Width, videoGenerationJobRequest.Height),
		videoGenerationJobRequest.NSeconds,
	)
	if !ok {
		return model.Price{}, errors.New("video output price not found")
	}

	return model.Price{
		VideoPerSecondPrice: model.ZeroNullFloat64(price),
	}, nil
}

func GetVideoGenerationJobRequestUsage(c *gin.Context, _ model.ModelConfig) (model.Usage, error) {
	videoGenerationJobRequest, err := utils.UnmarshalVideoGenerationJobRequest(c.Request)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		VideoSeconds:  model.ZeroNullInt64(videoGenerationJobRequest.NSeconds),
		VideoVariants: model.ZeroNullInt64(videoGenerationJobRequest.NVariants),
	}, nil
}
