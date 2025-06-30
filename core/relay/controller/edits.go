package controller

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
)

func GetImagesEditsRequestPrice(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	size := c.PostForm("size")
	quality := c.PostForm("quality")

	imageCostPrice, ok := GetImagesOutputPrice(mc, size, quality)
	if !ok {
		return model.Price{}, fmt.Errorf("invalid image size `%s` or quality `%s`", size, quality)
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

func GetImagesEditsRequestUsage(c *gin.Context, mc model.ModelConfig) (model.Usage, error) {
	mutliForms, err := c.MultipartForm()
	if err != nil {
		return model.Usage{}, err
	}

	images := int64(len(mutliForms.File["image"]))

	prompt := c.PostForm("prompt")
	nStr := c.PostForm("n")

	n := 1
	if nStr != "" {
		n, err = strconv.Atoi(nStr)
		if err != nil {
			return model.Usage{}, err
		}
	}

	return model.Usage{
		InputTokens: model.ZeroNullInt64(openai.CountTokenInput(
			prompt,
			mc.Model,
		)),
		ImageInputTokens: model.ZeroNullInt64(images),
		OutputTokens:     model.ZeroNullInt64(n),
	}, nil
}
