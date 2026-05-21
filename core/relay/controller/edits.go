package controller

import (
	"mime/multipart"
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
	if err := parseImagesEditsForm(c); err != nil {
		return err
	}

	if err := validateSupportedImageSize(c.PostForm("size"), mc); err != nil {
		return err
	}

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
	if len(mc.Price.ConditionalPrices) != 0 {
		price := mc.Price
		setImageOutputPriceUnit(&price, false)

		return price, nil
	}

	return model.Price{
		PerRequestPrice:     mc.Price.PerRequestPrice,
		InputPrice:          mc.Price.InputPrice,
		InputPriceUnit:      mc.Price.InputPriceUnit,
		ImageInputPrice:     mc.Price.ImageInputPrice,
		ImageInputPriceUnit: mc.Price.ImageInputPriceUnit,
		OutputPrice:         mc.Price.OutputPrice,
		OutputPriceUnit:     mc.Price.OutputPriceUnit,
	}, nil
}

func parseImagesEditsForm(c *gin.Context) error {
	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := common.ParseMultipartFormWithLimit(c.Request); err != nil {
			return NewBadRequestParamError(err.Error())
		}

		return nil
	}

	return NewBadRequestParamError("images edits requests must use multipart/form-data")
}

func GetImagesEditsRequestUsage(c *gin.Context, mc model.ModelConfig) (RequestUsage, error) {
	mutliForms, err := c.MultipartForm()
	if err != nil {
		return RequestUsage{}, err
	}

	images := countImagesEditFiles(mutliForms.File)

	prompt := c.PostForm("prompt")

	n := 1
	if parsedN, ok, err := getImagesEditsRequestN(c); err != nil {
		return RequestUsage{}, err
	} else if ok {
		n = parsedN
	}

	return RequestUsage{
		Usage: model.Usage{
			InputTokens: model.ZeroNullInt64(openai.CountTokenInput(
				prompt,
				mc.Model,
			)),
			ImageInputTokens: model.ZeroNullInt64(images),
			OutputTokens:     model.ZeroNullInt64(n),
		},
		Context: model.UsageContext{PriceCondition: model.UsagePriceCondition{
			Size:    c.PostForm("size"),
			Quality: c.PostForm("quality"),
		}},
	}, nil
}

func countImagesEditFiles(files map[string][]*multipart.FileHeader) int64 {
	return int64(len(files["image"]) + len(files["image[]"]))
}
