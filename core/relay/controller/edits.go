package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
)

func getImagesEditsRequestN(c *gin.Context) (int, bool, error) {
	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := common.ParseMultipartFormWithLimit(c.Request); err != nil {
			return 0, false, NewBadRequestParamError(err.Error())
		}
	} else if err := common.ParseFormWithLimit(c.Request); err != nil {
		return 0, false, NewBadRequestParamError(err.Error())
	}

	values, ok := c.Request.PostForm[imageRequestParamN]
	if !ok || len(values) == 0 {
		return 0, false, nil
	}

	if len(values) > 1 {
		return 0, true, NewBadRequestParamError("duplicate " + imageRequestParamN)
	}

	if values[0] == "" {
		return 0, false, nil
	}

	n, err := strconv.Atoi(values[0])
	if err != nil {
		return 0, true, NewBadRequestParamError("invalid " + imageRequestParamN)
	}

	return n, true, nil
}

func ValidateImagesEditsRequest(c *gin.Context, mc model.ModelConfig) error {
	n, ok, err := getImagesEditsRequestN(c)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	return validateImageGenerationCount(n, mc.MaxImageGenerationCount)
}

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

	n := 1
	if parsedN, ok, err := getImagesEditsRequestN(c); err != nil {
		return model.Usage{}, err
	} else if ok {
		n = parsedN
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
