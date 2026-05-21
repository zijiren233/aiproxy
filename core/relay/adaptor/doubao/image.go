package doubao

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/image"
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/render"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

const metaDoubaoImageResponseFormat = "doubao_image_response_format"

type doubaoImageResponse struct {
	Created int64                   `json:"created,omitempty"`
	Data    []*doubaoImageData      `json:"data,omitempty"`
	Usage   *doubaoImageUsage       `json:"usage,omitempty"`
	Error   *relaymodel.OpenAIError `json:"error,omitempty"`
}

type doubaoImageData struct {
	URL           string                  `json:"url,omitempty"`
	B64JSON       string                  `json:"b64_json,omitempty"`
	RevisedPrompt string                  `json:"revised_prompt,omitempty"`
	Size          string                  `json:"size,omitempty"`
	Error         *relaymodel.OpenAIError `json:"error,omitempty"`
}

type doubaoImageUsage struct {
	GeneratedImages int64                `json:"generated_images,omitempty"`
	OutputTokens    int64                `json:"output_tokens,omitempty"`
	TotalTokens     int64                `json:"total_tokens,omitempty"`
	ToolUsage       doubaoImageToolUsage `json:"tool_usage,omitempty"`
}

type doubaoImageToolUsage struct {
	WebSearch int64 `json:"web_search,omitempty"`
}

type doubaoImageStreamEvent struct {
	Type       string                  `json:"type,omitempty"`
	Model      string                  `json:"model,omitempty"`
	Created    int64                   `json:"created,omitempty"`
	ImageIndex int                     `json:"image_index,omitempty"`
	URL        string                  `json:"url,omitempty"`
	B64JSON    string                  `json:"b64_json,omitempty"`
	Size       string                  `json:"size,omitempty"`
	Usage      *doubaoImageUsage       `json:"usage,omitempty"`
	Error      *relaymodel.OpenAIError `json:"error,omitempty"`
}

type openAIImageStreamEvent struct {
	Type              string                  `json:"type"`
	B64JSON           string                  `json:"b64_json,omitempty"`
	URL               string                  `json:"url,omitempty"`
	CreatedAt         int64                   `json:"created_at,omitempty"`
	Size              string                  `json:"size,omitempty"`
	PartialImageIndex *int                    `json:"partial_image_index,omitempty"`
	Usage             *relaymodel.ImageUsage  `json:"usage,omitempty"`
	Error             *relaymodel.OpenAIError `json:"error,omitempty"`
}

func ConvertImageRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	node, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	responseFormat, err := node.Get("response_format").String()
	if err != nil && !errors.Is(err, ast.ErrNotExist) {
		return adaptor.ConvertResult{}, err
	}

	meta.Set(metaDoubaoImageResponseFormat, responseFormat)

	if err := normalizeDoubaoImageRequest(&node, meta.ActualModel); err != nil {
		return adaptor.ConvertResult{}, err
	}

	data, err := node.MarshalJSON()
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func normalizeDoubaoImageRequest(node *ast.Node, actualModel string) error {
	if _, err := node.Set("model", ast.NewString(actualModel)); err != nil {
		return err
	}

	if n, err := node.Get("n").Int64(); err == nil && n > 1 {
		if err := setDoubaoSequentialImages(node, n); err != nil {
			return err
		}
	}

	_, err := node.Unset("n")
	if err != nil && !errors.Is(err, ast.ErrNotExist) {
		return err
	}

	return nil
}

func setDoubaoSequentialImages(node *ast.Node, n int64) error {
	if n > 15 {
		n = 15
	}

	if _, err := node.Set("sequential_image_generation", ast.NewString("auto")); err != nil {
		return err
	}

	options := node.Get("sequential_image_generation_options")
	if options == nil || !options.Exists() || options.TypeSafe() == ast.V_NULL {
		optionsNode := ast.NewObject(nil)
		if _, err := node.Set("sequential_image_generation_options", optionsNode); err != nil {
			return err
		}

		options = node.Get("sequential_image_generation_options")
	}

	_, err := options.Set("max_images", ast.NewNumber(strconv.FormatInt(n, 10)))

	return err
}

func ImageHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := common.GetLogger(c)

	var response doubaoImageResponse
	if err := common.UnmarshalResponse(resp, &response); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	if response.Error != nil && response.Error.Message != "" {
		return adaptor.DoResponseResult{}, relaymodel.NewOpenAIError(
			http.StatusBadGateway,
			*response.Error,
		)
	}

	usage := doubaoImageUsageToModelUsage(response.Usage, meta.RequestUsage, response.Data)
	usageContext := doubaoImageUsageContext(response.Data).WithFallback(meta.RequestUsageContext)

	if meta.GetString(metaDoubaoImageResponseFormat) == "b64_json" {
		for _, data := range response.Data {
			if data == nil || data.B64JSON != "" || data.URL == "" {
				continue
			}

			var b64JSON string

			_, b64JSON, err := image.GetImageFromURL(c.Request.Context(), data.URL)
			if err != nil {
				log.Warnf("convert doubao image url to b64_json failed, keep original url: %v", err)
				continue
			}

			data.B64JSON = b64JSON
		}
	}

	data, err := sonic.Marshal(&response)
	if err != nil {
		return adaptor.DoResponseResult{Usage: usage, UsageContext: usageContext},
			relaymodel.WrapperOpenAIError(
				err,
				"marshal_response_body_failed",
				http.StatusInternalServerError,
			)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))

	if _, err := c.Writer.Write(data); err != nil {
		log.Warnf("write response body failed: %v", err)
	}

	return adaptor.DoResponseResult{Usage: usage, UsageContext: usageContext}, nil
}

func ImageStreamHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, openai.ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := common.GetLogger(c)

	scanner, cleanup := relayutils.NewStreamScanner(resp.Body, meta.ActualModel)
	defer cleanup()

	usage := coremodel.Usage{}
	usageContext := meta.RequestUsageContext

	for scanner.Scan() {
		line := scanner.Bytes()
		if !render.IsValidSSEData(line) {
			continue
		}

		data := render.ExtractSSEData(line)
		if render.IsSSEDone(data) {
			render.OpenaiBytesData(c, data)
			continue
		}

		var event doubaoImageStreamEvent
		if err := sonic.Unmarshal(data, &event); err != nil {
			log.Errorf("error unmarshalling doubao image stream response: %v", err)
			render.OpenaiBytesData(c, data)
			continue
		}

		openAIEvent, eventUsage, eventContext := convertDoubaoImageStreamEvent(
			event,
			meta.RequestUsage,
		)
		if eventUsage != nil {
			usage = *eventUsage
		}

		usageContext = eventContext.WithFallback(usageContext)

		if err := render.OpenaiObjectData(c, openAIEvent); err != nil {
			log.Errorf("write doubao image stream response failed: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("error reading doubao image stream: %v", err)
	}

	return adaptor.DoResponseResult{Usage: usage, UsageContext: usageContext}, nil
}

func convertDoubaoImageStreamEvent(
	event doubaoImageStreamEvent,
	requestUsage coremodel.Usage,
) (openAIImageStreamEvent, *coremodel.Usage, coremodel.UsageContext) {
	openAIEvent := openAIImageStreamEvent{
		Type:      event.Type,
		B64JSON:   event.B64JSON,
		URL:       event.URL,
		CreatedAt: event.Created,
		Size:      normalizeDoubaoImageSize(event.Size),
		Error:     event.Error,
	}

	usageContext := coremodel.UsageContext{}
	if openAIEvent.Size != "" {
		usageContext.PriceCondition.Size = openAIEvent.Size
	}

	switch event.Type {
	case "image_generation.partial_succeeded":
		index := event.ImageIndex
		openAIEvent.Type = "image_generation.partial_image"
		openAIEvent.PartialImageIndex = &index
	case "image_generation.completed":
		openAIEvent.Type = "image_generation.completed"
		usage := doubaoImageUsageToModelUsage(event.Usage, requestUsage, nil)
		openAIEvent.Usage = doubaoImageUsageToOpenAIUsage(usage)
		return openAIEvent, &usage, usageContext
	}

	return openAIEvent, nil, usageContext
}

func doubaoImageUsageToOpenAIUsage(usage coremodel.Usage) *relaymodel.ImageUsage {
	return &relaymodel.ImageUsage{
		InputTokens:  int64(usage.InputTokens),
		OutputTokens: int64(usage.OutputTokens),
		TotalTokens:  int64(usage.TotalTokens),
		InputTokensDetails: relaymodel.ImageInputTokensDetails{
			ImageTokens: int64(usage.ImageInputTokens),
		},
		OutputTokensDetails: &relaymodel.ImageOutputTokensDetails{
			ImageTokens: int64(usage.ImageOutputTokens),
		},
	}
}

func doubaoImageUsageToModelUsage(
	usage *doubaoImageUsage,
	requestUsage coremodel.Usage,
	data []*doubaoImageData,
) coremodel.Usage {
	if usage == nil {
		output := int64(0)
		for _, item := range data {
			if item != nil && item.Error == nil && (item.URL != "" || item.B64JSON != "") {
				output++
			}
		}

		if output == 0 {
			output = int64(requestUsage.OutputTokens)
		}

		return coremodel.Usage{
			OutputTokens:      coremodel.ZeroNullInt64(output),
			ImageOutputTokens: coremodel.ZeroNullInt64(output),
			TotalTokens:       coremodel.ZeroNullInt64(output),
		}
	}

	outputTokens := usage.GeneratedImages
	if outputTokens == 0 {
		outputTokens = countSuccessfulDoubaoImages(data)
	}

	if outputTokens == 0 {
		outputTokens = int64(requestUsage.OutputTokens)
	}

	totalTokens := outputTokens

	return coremodel.Usage{
		OutputTokens:      coremodel.ZeroNullInt64(outputTokens),
		ImageOutputTokens: coremodel.ZeroNullInt64(outputTokens),
		TotalTokens:       coremodel.ZeroNullInt64(totalTokens),
		WebSearchCount:    coremodel.ZeroNullInt64(usage.ToolUsage.WebSearch),
	}
}

func doubaoImageUsageContext(data []*doubaoImageData) coremodel.UsageContext {
	for _, item := range data {
		if item == nil || item.Size == "" {
			continue
		}

		return coremodel.UsageContext{
			PriceCondition: coremodel.UsagePriceCondition{
				Size: normalizeDoubaoImageSize(item.Size),
			},
		}
	}

	return coremodel.UsageContext{}
}

func countSuccessfulDoubaoImages(data []*doubaoImageData) int64 {
	output := int64(0)
	for _, item := range data {
		if item != nil && item.Error == nil && (item.URL != "" || item.B64JSON != "") {
			output++
		}
	}

	return output
}

func normalizeDoubaoImageSize(size string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(size), "×", "x"))
}
