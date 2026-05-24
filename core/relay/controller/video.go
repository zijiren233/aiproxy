package controller

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

const (
	defaultGeminiVideoDurationSeconds = 8
	defaultGeminiVideoResolution      = "720p"
)

type geminiVideoRequestUsageParams struct {
	seconds    int
	variants   int
	resolution string
}

type videosRequestUsageParams struct {
	seconds int
	size    string
}

func ValidateVideoGenerationJobRequest(c *gin.Context, mc model.ModelConfig) error {
	request, err := getVideoGenerationJobRequest(c)
	if err != nil {
		return err
	}

	return validateVideoGenerationJobRequest(request, mc)
}

func ValidateVideosRequest(c *gin.Context, mc model.ModelConfig) error {
	params, err := getVideosRequestUsageParams(c)
	if err != nil {
		return err
	}

	return validateVideosRequestUsageParams(params, mc)
}

func ValidateGeminiVideoRequest(c *gin.Context, mc model.ModelConfig) error {
	params, err := getGeminiVideoRequestUsageParams(c)
	if err != nil {
		return err
	}

	return validateGeminiVideoRequestUsageParams(params, mc)
}

func GetVideoGenerationJobRequestPrice(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	request, err := getVideoGenerationJobRequest(c)
	if err != nil {
		return model.Price{}, err
	}

	if err := validateVideoGenerationJobRequest(request, mc); err != nil {
		return model.Price{}, err
	}

	return getVideoRequestPrice(mc.Price), nil
}

func GetGeminiVideoRequestPrice(_ *gin.Context, mc model.ModelConfig) (model.Price, error) {
	return getVideoRequestPrice(mc.Price), nil
}

func GetVideosRequestPrice(c *gin.Context, mc model.ModelConfig) (model.Price, error) {
	params, err := getVideosRequestUsageParams(c)
	if err != nil {
		return model.Price{}, err
	}

	if err := validateVideosRequestUsageParams(params, mc); err != nil {
		return model.Price{}, err
	}

	return getVideoRequestPrice(mc.Price), nil
}

func getVideoRequestPrice(price model.Price) model.Price {
	setVideoOutputPriceUnit(&price, false)
	return price
}

func validateVideoGenerationSeconds(seconds, maxSeconds int) error {
	if maxSeconds <= 0 || seconds <= maxSeconds {
		return nil
	}

	return NewBadRequestParamError(
		fmt.Sprintf("seconds must be less than or equal to %d", maxSeconds),
	)
}

func validateVideoGenerationCount(count, maxCount int) error {
	if maxCount <= 0 || count <= maxCount {
		return nil
	}

	return NewBadRequestParamError(
		fmt.Sprintf("video count must be less than or equal to %d", maxCount),
	)
}

func validateOpenAIVideoSizeFormat(size string) error {
	size = strings.ToLower(strings.TrimSpace(size))
	if size == "" || dimensionResolutionValue(size) {
		return nil
	}

	return NewBadRequestParamError(fmt.Sprintf("invalid video size `%s`", size))
}

func validateOpenAIVideoJobResolutionFormat(
	request *relaymodel.VideoGenerationJobRequest,
) error {
	if request == nil {
		return nil
	}

	if request.Size != "" {
		return validateOpenAIVideoSizeFormat(request.Size)
	}

	if (request.Width == 0) != (request.Height == 0) {
		return NewBadRequestParamError("width and height must be provided together")
	}

	return nil
}

func validateGeminiNativeVideoResolutionFormat(resolution string) error {
	resolution = normalizeSupportedResolutionValue(resolution)
	switch resolution {
	case "", "720p", "1080p", "4k":
		return nil
	default:
		return NewBadRequestParamError(fmt.Sprintf("invalid gemini video resolution `%s`", resolution))
	}
}

func validateSupportedVideoResolution(resolution string, mc model.ModelConfig) error {
	if supportedResolutionMatches(
		resolution,
		mc.AllowedResolutions,
		videoResolutionAliases,
		videoResolutionAliases,
		!mc.DisableResolutionFuzzyMatch,
	) {
		return nil
	}

	return NewBadRequestParamError(fmt.Sprintf("unsupported video resolution `%s`", resolution))
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

	if err := validateVideoGenerationJobRequest(request, mc); err != nil {
		return RequestUsage{}, err
	}

	return RequestUsage{
		// Video providers bill with incompatible units: returned videos, seconds,
		// resolution tiers, or provider-specific async usage. Do not reserve
		// balance from a preflight estimate; response/async handlers provide the
		// final billable usage after the upstream result is known.
		Usage: model.Usage{},
		Context: model.UsageContext{
			Resolution: videoRequestPriceResolution(request),
		},
	}, nil
}

func GetGeminiVideoRequestUsage(c *gin.Context, mc model.ModelConfig) (RequestUsage, error) {
	params, err := getGeminiVideoRequestUsageParams(c)
	if err != nil {
		return RequestUsage{}, err
	}

	if err := validateGeminiVideoRequestUsageParams(params, mc); err != nil {
		return RequestUsage{}, err
	}

	return RequestUsage{
		// Native Gemini/Veo polling does not expose final duration or resolution
		// metadata, and OpenAI-compatible video models use different units. Keep
		// preflight usage empty; the Gemini adaptor stores request-side metadata
		// for async completion billing without mutating RequestUsage.
		Usage: model.Usage{},
		Context: model.UsageContext{
			Resolution: params.resolution,
		},
	}, nil
}

func GetVideosRequestUsage(c *gin.Context, mc model.ModelConfig) (RequestUsage, error) {
	params, err := getVideosRequestUsageParams(c)
	if err != nil {
		return RequestUsage{}, err
	}

	if err := validateVideosRequestUsageParams(params, mc); err != nil {
		return RequestUsage{}, err
	}

	return RequestUsage{
		// Video usage is provider-specific and often async. Do not use requested
		// seconds as a preflight balance estimate; final usage is supplied by the
		// response or async usage fetcher.
		Usage: model.Usage{},
		Context: model.UsageContext{
			Resolution: params.size,
		},
	}, nil
}

func validateVideoGenerationJobRequest(
	request *relaymodel.VideoGenerationJobRequest,
	mc model.ModelConfig,
) error {
	if err := validateOpenAIVideoJobResolutionFormat(request); err != nil {
		return err
	}

	if err := validateVideoGenerationSeconds(
		request.NSeconds,
		mc.MaxVideoGenerationSeconds,
	); err != nil {
		return err
	}

	if err := validateVideoGenerationCount(
		request.NVariants,
		mc.MaxVideoGenerationCount,
	); err != nil {
		return err
	}

	return validateSupportedVideoResolution(videoRequestPriceResolution(request), mc)
}

func getVideosRequestUsageParams(c *gin.Context) (videosRequestUsageParams, error) {
	if strings.HasPrefix(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		if err := common.ParseMultipartFormWithLimit(c.Request); err != nil {
			return videosRequestUsageParams{}, NewBadRequestParamError(err.Error())
		}

		seconds, err := parseOptionalPositiveInt(c.PostForm("seconds"), "seconds")
		if err != nil {
			return videosRequestUsageParams{}, err
		}

		return videosRequestUsageParams{
			seconds: seconds,
			size:    c.PostForm("size"),
		}, nil
	}

	node, err := common.UnmarshalRequest2NodeReusable(c.Request)
	if err != nil {
		return videosRequestUsageParams{}, NewBadRequestParamError(err.Error())
	}

	seconds, _, err := intValueFromNode(&node, "seconds")
	if err != nil {
		return videosRequestUsageParams{}, err
	}

	return videosRequestUsageParams{
		seconds: seconds,
		size:    stringValueFromNode(&node, "size"),
	}, nil
}

func validateVideosRequestUsageParams(params videosRequestUsageParams, mc model.ModelConfig) error {
	if err := validateOpenAIVideoSizeFormat(params.size); err != nil {
		return err
	}

	if err := validateVideoGenerationSeconds(
		params.seconds,
		mc.MaxVideoGenerationSeconds,
	); err != nil {
		return err
	}

	return validateSupportedVideoResolution(params.size, mc)
}

func getGeminiVideoRequestUsageParams(c *gin.Context) (geminiVideoRequestUsageParams, error) {
	node, err := common.UnmarshalRequest2NodeReusable(c.Request)
	if err != nil {
		return geminiVideoRequestUsageParams{}, NewBadRequestParamError(err.Error())
	}

	parameters := node.Get("parameters")
	params := geminiVideoRequestUsageParams{
		seconds:    defaultGeminiVideoDurationSeconds,
		variants:   1,
		resolution: defaultGeminiVideoResolution,
	}

	parsedResolution := geminiVideoStringValueFromNode(&node, parameters, "resolution")
	if parsedResolution != "" {
		params.resolution = parsedResolution
	}

	parsedSeconds, ok, err := geminiVideoIntValueFromNode(&node, parameters, "durationSeconds")
	if err != nil {
		return geminiVideoRequestUsageParams{}, err
	}

	if ok && parsedSeconds > 0 {
		params.seconds = parsedSeconds
	}

	parsedVariants, ok, err := geminiVideoIntValueFromNode(&node, parameters, "numberOfVideos")
	if err != nil {
		return geminiVideoRequestUsageParams{}, err
	}

	if ok && parsedVariants > 0 {
		params.variants = parsedVariants
	}

	return params, nil
}

func geminiVideoIntValueFromNode(
	node *ast.Node,
	parameters *ast.Node,
	name string,
) (int, bool, error) {
	value, ok, err := intValueFromNode(node, name)
	if err != nil || (ok && value != 0) {
		return value, ok, err
	}

	parameterValue, parameterOK, err := intValueFromNode(parameters, name)
	if err != nil || parameterOK {
		return parameterValue, parameterOK, err
	}

	return value, ok, nil
}

func geminiVideoStringValueFromNode(
	node *ast.Node,
	parameters *ast.Node,
	name string,
) string {
	if value := stringValueFromNode(node, name); value != "" {
		return value
	}

	return stringValueFromNode(parameters, name)
}

func validateGeminiVideoRequestUsageParams(
	params geminiVideoRequestUsageParams,
	mc model.ModelConfig,
) error {
	if err := validateGeminiNativeVideoResolutionFormat(params.resolution); err != nil {
		return err
	}

	if err := validateVideoGenerationSeconds(
		params.seconds,
		mc.MaxVideoGenerationSeconds,
	); err != nil {
		return err
	}

	if err := validateVideoGenerationCount(
		params.variants,
		mc.MaxVideoGenerationCount,
	); err != nil {
		return err
	}

	return validateSupportedVideoResolution(params.resolution, mc)
}

func getVideoGenerationJobRequest(c *gin.Context) (*relaymodel.VideoGenerationJobRequest, error) {
	contentType := c.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := common.ParseMultipartFormWithLimit(c.Request); err != nil {
			return nil, NewBadRequestParamError(err.Error())
		}

		return getMultipartVideoGenerationJobRequest(c)
	}

	request, err := utils.UnmarshalVideoGenerationJobRequest(c.Request)
	if err != nil {
		return nil, err
	}

	if err := validateParsedVideoGenerationJobRequest(request); err != nil {
		return nil, err
	}

	return request, nil
}

func intValueFromNode(node *ast.Node, name string) (int, bool, error) {
	if node == nil || !node.Exists() || node.TypeSafe() == ast.V_NULL {
		return 0, false, nil
	}

	valueNode := node.Get(name)
	if valueNode == nil || !valueNode.Exists() || valueNode.TypeSafe() == ast.V_NULL {
		return 0, false, nil
	}

	if valueNode.TypeSafe() == ast.V_STRING {
		value, err := valueNode.String()
		if err != nil {
			return 0, true, NewBadRequestParamError(
				fmt.Sprintf("invalid %s: %s", name, err.Error()),
			)
		}

		parsed, err := parseOptionalPositiveInt(value, name)
		if err != nil {
			return 0, true, err
		}

		return parsed, true, nil
	}

	value, err := valueNode.Int64()
	if err != nil {
		return 0, true, NewBadRequestParamError(
			fmt.Sprintf("invalid %s: %s", name, err.Error()),
		)
	}

	if value < 0 {
		return 0, true, NewBadRequestParamError(
			fmt.Sprintf("invalid %s: must be non-negative", name),
		)
	}

	return int(value), true, nil
}

func stringValueFromNode(node *ast.Node, name string) string {
	if node == nil || !node.Exists() || node.TypeSafe() == ast.V_NULL {
		return ""
	}

	valueNode := node.Get(name)
	if valueNode == nil || !valueNode.Exists() || valueNode.TypeSafe() == ast.V_NULL {
		return ""
	}

	value, err := valueNode.String()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(value)
}

func validateParsedVideoGenerationJobRequest(request *relaymodel.VideoGenerationJobRequest) error {
	if request.NSeconds < 0 {
		return NewBadRequestParamError("invalid n_seconds: must be non-negative")
	}

	if request.NVariants < 0 {
		return NewBadRequestParamError("invalid n_variants: must be non-negative")
	}

	return nil
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

	return request, nil
}

func parseOptionalPositiveInt(value, name string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewBadRequestParamError(
			fmt.Sprintf("invalid %s: %s", name, err.Error()),
		)
	}

	if parsed < 0 {
		return 0, NewBadRequestParamError(fmt.Sprintf("invalid %s: must be non-negative", name))
	}

	return parsed, nil
}

func videoRequestPriceResolution(request *relaymodel.VideoGenerationJobRequest) string {
	if request == nil {
		return ""
	}

	if request.Size != "" {
		return request.Size
	}

	if request.Width > 0 && request.Height > 0 {
		return relaymodel.VideoResolutionFromDimensions(request.Width, request.Height)
	}

	return ""
}
