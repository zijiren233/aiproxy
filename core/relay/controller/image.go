package controller

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

const imageRequestParamN = "n"

type RequestParamError struct {
	StatusCode int
	Message    string
}

func (e *RequestParamError) Error() string {
	return e.Message
}

func NewBadRequestParamError(message string) error {
	return &RequestParamError{
		StatusCode: http.StatusBadRequest,
		Message:    message,
	}
}

func validateImageGenerationCount(n, maxCount int) error {
	if maxCount <= 0 || n <= maxCount {
		return nil
	}

	return NewBadRequestParamError(
		fmt.Sprintf("%s must be less than or equal to %d", imageRequestParamN, maxCount),
	)
}

func getImagesRequestN(c *gin.Context) (int, bool, error) {
	node, err := common.UnmarshalRequest2NodeReusable(c.Request)
	if err != nil {
		return 0, false, err
	}

	iter, err := node.Properties()
	if err != nil {
		return 0, false, err
	}

	count := 0

	var pair ast.Pair
	for iter.Next(&pair) {
		if pair.Key != imageRequestParamN {
			continue
		}

		count++
		if count > 1 {
			return 0, true, NewBadRequestParamError("duplicate " + imageRequestParamN)
		}
	}

	nNode := node.Get(imageRequestParamN)
	if nNode == nil || !nNode.Exists() || nNode.TypeSafe() == ast.V_NULL {
		return 0, false, nil
	}

	n, err := nNode.Int64()
	if err != nil {
		return 0, true, NewBadRequestParamError("invalid " + imageRequestParamN)
	}

	return int(n), true, nil
}

func getImagesRequest(c *gin.Context) (*relaymodel.ImageRequest, error) {
	imageRequest, err := utils.UnmarshalImageRequest(c.Request)
	if err != nil {
		return nil, err
	}

	if imageRequest.Prompt == "" {
		return nil, errors.New("prompt is required")
	}

	if imageRequest.N == 0 {
		imageRequest.N = 1
	}

	return imageRequest, nil
}

func ValidateImagesRequest(c *gin.Context, mc model.ModelConfig) error {
	n, ok, err := getImagesRequestN(c)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	return validateImageGenerationCount(n, mc.MaxImageGenerationCount)
}

func GetImagesOutputPrice(modelConfig model.ModelConfig, size, quality string) (float64, bool) {
	switch {
	case len(modelConfig.ImagePrices) == 0 && len(modelConfig.ImageQualityPrices) == 0:
		return float64(modelConfig.Price.OutputPrice), true
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

func GetImagesRequestPrice(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	imageRequest, err := getImagesRequest(c)
	if err != nil {
		return model.Price{}, err
	}

	imageCostPrice, ok := GetImagesOutputPrice(mc, imageRequest.Size, imageRequest.Quality)
	if !ok {
		return model.Price{}, fmt.Errorf(
			"invalid image size `%s` or quality `%s`",
			imageRequest.Size,
			imageRequest.Quality,
		)
	}

	return model.Price{
		PerRequestPrice:     mc.Price.PerRequestPrice,
		InputPrice:          mc.Price.InputPrice,
		InputPriceUnit:      mc.Price.InputPriceUnit,
		ImageInputPrice:     mc.Price.ImageInputPrice,
		ImageInputPriceUnit: mc.Price.ImageInputPriceUnit,
		OutputPrice:         model.ZeroNullFloat64(imageCostPrice),
		OutputPriceUnit:     mc.Price.OutputPriceUnit,
	}, nil
}

func GetImagesRequestUsage(c *gin.Context, _ model.ModelConfig) (model.Usage, error) {
	imageRequest, err := getImagesRequest(c)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		InputTokens: model.ZeroNullInt64(openai.CountTokenInput(
			imageRequest.Prompt,
			imageRequest.Model,
		)),
		OutputTokens: model.ZeroNullInt64(imageRequest.N),
	}, nil
}
