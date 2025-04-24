package controller

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func getImageRequest(c *gin.Context) (*relaymodel.ImageRequest, error) {
	imageRequest, err := utils.UnmarshalImageRequest(c.Request)
	if err != nil {
		return nil, err
	}
	if imageRequest.Prompt == "" {
		return nil, errors.New("prompt is required")
	}
	if imageRequest.Size == "" {
		return nil, errors.New("size is required")
	}
	if imageRequest.N == 0 {
		imageRequest.N = 1
	}
	return imageRequest, nil
}

func GetImageOutputPrice(modelConfig *model.ModelConfig, size string, quality string) (float64, bool) {
	switch {
	case len(modelConfig.ImagePrices) == 0 && len(modelConfig.ImageQualityPrices) == 0:
		return modelConfig.Price.OutputPrice, true
	case len(modelConfig.ImageQualityPrices) != 0:
		price, ok := modelConfig.ImageQualityPrices[size][quality]
		return price, ok
	case len(modelConfig.ImagePrices) != 0:
		price, ok := modelConfig.ImagePrices[size]
		return price, ok
	default:
		return 0, false
	}
}

func GetImageRequestPrice(c *gin.Context, mc *model.ModelConfig) (model.Price, error) {
	imageRequest, err := getImageRequest(c)
	if err != nil {
		return model.Price{}, err
	}

	imageCostPrice, ok := GetImageOutputPrice(mc, imageRequest.Size, imageRequest.Quality)
	if !ok {
		return model.Price{}, fmt.Errorf("invalid image size `%s` or quality `%s`", imageRequest.Size, imageRequest.Quality)
	}

	return model.Price{
		PerRequestPrice:      mc.Price.PerRequestPrice,
		InputPrice:           mc.Price.InputPrice,
		InputPriceUnit:       mc.Price.InputPriceUnit,
		ImageInputPrice:      mc.Price.ImageInputPrice,
		ImageInputPriceUnit:  mc.Price.ImageInputPriceUnit,
		OutputPrice:          mc.Price.OutputPrice,
		OutputPriceUnit:      mc.Price.OutputPriceUnit,
		ImageOutputPrice:     imageCostPrice,
		ImageOutputPriceUnit: mc.Price.ImageOutputPriceUnit,
	}, nil
}

func GetImageRequestUsage(c *gin.Context, _ *model.ModelConfig) (model.Usage, error) {
	imageRequest, err := getImageRequest(c)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		InputTokens:        openai.CountTokenInput(imageRequest.Prompt, imageRequest.Model),
		ImageOutputNumbers: int64(imageRequest.N),
	}, nil
}
