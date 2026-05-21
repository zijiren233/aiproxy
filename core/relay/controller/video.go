package controller

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func GetVideoGenerationJobRequestPrice(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	request, err := getVideoGenerationJobRequest(c)
	if err != nil {
		return model.Price{}, err
	}

	if err := validateVideoGenerationSeconds(
		request.NSeconds,
		mc.MaxVideoGenerationSeconds,
	); err != nil {
		return model.Price{}, err
	}

	if err := validateSupportedVideoSize(videoRequestPriceSize(request), mc); err != nil {
		return model.Price{}, err
	}

	price := mc.Price
	setVideoOutputPriceUnit(&price, false)

	return price, nil
}

func validateVideoGenerationSeconds(seconds, maxSeconds int) error {
	if maxSeconds <= 0 || seconds <= maxSeconds {
		return nil
	}

	return NewBadRequestParamError(
		fmt.Sprintf("seconds must be less than or equal to %d", maxSeconds),
	)
}

func validateSupportedVideoSize(size string, mc model.ModelConfig) error {
	if size == "" {
		return nil
	}

	sizes, ok := model.GetModelConfigStringSlice(mc.Config, model.ModelConfigVideoSizes)
	if !ok || len(sizes) == 0 {
		return nil
	}

	if slices.Contains(sizes, size) {
		return nil
	}

	return NewBadRequestParamError(fmt.Sprintf("unsupported video size `%s`", size))
}

func setVideoOutputPriceUnit(price *model.Price, force bool) {
	if price == nil {
		return
	}

	if (force || len(price.ConditionalPrices) != 0) && price.OutputPriceUnit == 0 {
		price.OutputPriceUnit = 1
	}

	for i := range price.ConditionalPrices {
		setVideoOutputPriceUnit(&price.ConditionalPrices[i].Price, true)
	}
}

func GetVideoGenerationJobRequestUsage(c *gin.Context, mc model.ModelConfig) (RequestUsage, error) {
	request, err := getVideoGenerationJobRequest(c)
	if err != nil {
		return RequestUsage{}, err
	}

	priceSize := videoRequestPriceSize(request)
	if err := validateSupportedVideoSize(priceSize, mc); err != nil {
		return RequestUsage{}, err
	}

	if err := validateVideoGenerationSeconds(
		request.NSeconds,
		mc.MaxVideoGenerationSeconds,
	); err != nil {
		return RequestUsage{}, err
	}

	seconds := videoRequestSeconds(request)
	if seconds <= 0 {
		return RequestUsage{}, nil
	}

	return RequestUsage{
		Usage: model.Usage{
			OutputTokens: model.ZeroNullInt64(seconds),
			TotalTokens:  model.ZeroNullInt64(seconds),
		},
		Context: model.UsageContext{
			PriceCondition: model.UsagePriceCondition{Size: priceSize},
		},
	}, nil
}

func getVideoGenerationJobRequest(c *gin.Context) (*relaymodel.VideoGenerationJobRequest, error) {
	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := common.ParseMultipartFormWithLimit(c.Request); err != nil {
			return nil, err
		}

		return getMultipartVideoGenerationJobRequest(c)
	}

	request, err := utils.UnmarshalVideoGenerationJobRequest(c.Request)
	if err != nil {
		return nil, err
	}

	if request.NSeconds == 0 {
		if seconds, ok, err := intValueFromReusableRequest(c, "seconds"); err != nil {
			return nil, err
		} else if ok {
			request.NSeconds = seconds
		}
	}

	if err := validateParsedVideoGenerationJobRequest(request); err != nil {
		return nil, err
	}

	return request, nil
}

func validateParsedVideoGenerationJobRequest(request *relaymodel.VideoGenerationJobRequest) error {
	if request.NSeconds < 0 {
		return errors.New("invalid n_seconds: must be non-negative")
	}

	if request.NVariants < 0 {
		return errors.New("invalid n_variants: must be non-negative")
	}

	return nil
}

func intValueFromReusableRequest(c *gin.Context, name string) (int, bool, error) {
	node, err := common.UnmarshalRequest2NodeReusable(c.Request)
	if err != nil {
		return 0, false, err
	}

	valueNode := node.Get(name)
	if valueNode == nil || !valueNode.Exists() || valueNode.TypeSafe() == ast.V_NULL {
		return 0, false, nil
	}

	if valueNode.TypeSafe() == ast.V_STRING {
		value, err := valueNode.String()
		if err != nil {
			return 0, true, fmt.Errorf("invalid %s: %w", name, err)
		}

		parsed, err := parseOptionalPositiveInt(value, name)
		if err != nil {
			return 0, true, err
		}

		return parsed, true, nil
	}

	value, err := valueNode.Int64()
	if err != nil {
		return 0, true, fmt.Errorf("invalid %s: %w", name, err)
	}

	if value < 0 {
		return 0, true, fmt.Errorf("invalid %s: must be non-negative", name)
	}

	return int(value), true, nil
}

func getMultipartVideoGenerationJobRequest(
	c *gin.Context,
) (*relaymodel.VideoGenerationJobRequest, error) {
	request := &relaymodel.VideoGenerationJobRequest{
		Prompt: c.PostForm("prompt"),
		Model:  c.PostForm("model"),
		Size:   c.PostForm("size"),
	}

	var err error
	if request.Width, err = parseOptionalPositiveInt(c.PostForm("width"), "width"); err != nil {
		return nil, err
	}

	if request.Height, err = parseOptionalPositiveInt(c.PostForm("height"), "height"); err != nil {
		return nil, err
	}

	request.NVariants, err = parseOptionalPositiveInt(c.PostForm("n_variants"), "n_variants")
	if err != nil {
		return nil, err
	}

	request.NSeconds, err = parseOptionalPositiveInt(c.PostForm("n_seconds"), "n_seconds")
	if err != nil {
		return nil, err
	}

	if seconds, err := parseOptionalPositiveInt(c.PostForm("seconds"), "seconds"); err != nil {
		return nil, err
	} else if request.NSeconds == 0 {
		request.NSeconds = seconds
	}

	return request, nil
}

func parseOptionalPositiveInt(value, name string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}

	if parsed < 0 {
		return 0, fmt.Errorf("invalid %s: must be non-negative", name)
	}

	return parsed, nil
}

func videoRequestSeconds(request *relaymodel.VideoGenerationJobRequest) int64 {
	if request == nil {
		return 0
	}

	seconds := request.NSeconds
	if seconds <= 0 {
		return 0
	}

	variants := request.NVariants
	if variants == 0 {
		variants = 1
	}

	return int64(seconds * variants)
}

func videoRequestPriceSize(request *relaymodel.VideoGenerationJobRequest) string {
	if request == nil {
		return ""
	}

	if request.Size != "" {
		return request.Size
	}

	if request.Width > 0 && request.Height > 0 {
		return relaymodel.VideoPriceSizeFromDimensions(request.Width, request.Height)
	}

	return ""
}
