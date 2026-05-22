package gemini

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/render"
	"github.com/labring/aiproxy/core/relay/utils"
)

func ConvertImageRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	imageRequest, err := utils.UnmarshalImageRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if meta != nil {
		meta.Set(openai.MetaResponseFormat, imageRequest.ResponseFormat)
		meta.Set("stream", imageRequest.Stream)
	}

	if imageRequest.Prompt == "" {
		return adaptor.ConvertResult{}, errors.New("prompt is required")
	}

	geminiRequest := relaymodel.GeminiChatRequest{
		Contents: []*relaymodel.GeminiChatContent{
			{
				Role: relaymodel.GeminiRoleUser,
				Parts: []*relaymodel.GeminiPart{
					{Text: imageRequest.Prompt},
				},
			},
		},
		GenerationConfig: buildImageGenerationConfig(imageRequest),
	}

	data, err := sonic.Marshal(geminiRequest)
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

func buildImageGenerationConfig(
	imageRequest *relaymodel.ImageRequest,
) *relaymodel.GeminiChatGenerationConfig {
	config := &relaymodel.GeminiChatGenerationConfig{
		ResponseModalities: []string{
			relaymodel.GeminiModalityText,
			relaymodel.GeminiModalityImage,
		},
		ImageConfig: &relaymodel.GeminiImageConfig{},
	}

	if aspectRatio := geminiImageAspectRatioFromSize(imageRequest.Size); aspectRatio != "" {
		config.ImageConfig.AspectRatio = aspectRatio
	}

	if imageSize := geminiImageSizeFromSize(imageRequest.Size); imageSize != "" {
		config.ImageConfig.ImageSize = imageSize
	}

	if config.ImageConfig.AspectRatio == "" && config.ImageConfig.ImageSize == "" {
		config.ImageConfig = nil
	}

	return config
}

func geminiImageAspectRatioFromSize(size string) string {
	size = strings.ToLower(strings.TrimSpace(size))
	switch size {
	case "1:1", "3:4", "4:3", "9:16", "16:9":
		return size
	}

	width, height, ok := parseGeminiVideoDimensions(size)
	if !ok || width <= 0 || height <= 0 {
		return ""
	}

	return closestGeminiImageAspectRatio(width, height)
}

func geminiImageSizeFromSize(size string) string {
	size = strings.ToLower(strings.TrimSpace(size))
	switch size {
	case "512", "1k", "2k", "4k":
		return size
	default:
		return ""
	}
}

func closestGeminiImageAspectRatio(width, height int) string {
	type candidate struct {
		label string
		ratio float64
	}

	ratio := float64(width) / float64(height)
	candidates := []candidate{
		{"1:1", 1},
		{"1:4", 1.0 / 4.0},
		{"1:8", 1.0 / 8.0},
		{"2:3", 2.0 / 3.0},
		{"3:2", 3.0 / 2.0},
		{"3:4", 3.0 / 4.0},
		{"4:3", 4.0 / 3.0},
		{"4:1", 4.0},
		{"4:5", 4.0 / 5.0},
		{"5:4", 5.0 / 4.0},
		{"8:1", 8.0},
		{"9:16", 9.0 / 16.0},
		{"16:9", 16.0 / 9.0},
		{"21:9", 21.0 / 9.0},
	}

	best := candidates[0]
	bestDelta := absFloat(ratio - best.ratio)

	for _, item := range candidates[1:] {
		delta := absFloat(ratio - item.ratio)
		if delta < bestDelta {
			best = item
			bestDelta = delta
		}
	}

	return best.label
}

func ImageHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	if utils.IsStreamResponse(resp) {
		return imageStreamHandler(meta, c, resp)
	}

	defer resp.Body.Close()

	var geminiResponse relaymodel.GeminiChatResponse
	if err := sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&geminiResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	imageResponse, usage := geminiImageResponseToOpenAI(meta, &geminiResponse)
	if len(imageResponse.Data) == 0 {
		return adaptor.DoResponseResult{Usage: usage}, relaymodel.WrapperOpenAIErrorWithMessage(
			"gemini image response image is empty",
			"empty_image",
			http.StatusInternalServerError,
		)
	}

	data, err := sonic.Marshal(imageResponse)
	if err != nil {
		return adaptor.DoResponseResult{Usage: usage}, relaymodel.WrapperOpenAIError(
			err,
			"marshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = c.Writer.Write(data)

	return adaptor.DoResponseResult{Usage: usage}, nil
}

func geminiImageResponseToOpenAI(
	meta *meta.Meta,
	response *relaymodel.GeminiChatResponse,
) (relaymodel.ImageResponse, model.Usage) {
	imageResponse := relaymodel.ImageResponse{
		Created: time.Now().Unix(),
		Data:    make([]*relaymodel.ImageData, 0),
	}

	for _, candidate := range response.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData == nil ||
				!strings.HasPrefix(part.InlineData.MimeType, "image/") {
				continue
			}

			imageResponse.Data = append(imageResponse.Data, &relaymodel.ImageData{
				B64Json: part.InlineData.Data,
			})
		}
	}

	usage := geminiImageUsageFromResponse(
		meta,
		response.UsageMetadata,
		int64(len(imageResponse.Data)),
	)
	imageResponse.Usage = usageToImageUsagePtr(usage)

	return imageResponse, usage
}

func geminiImageUsageFromResponse(
	meta *meta.Meta,
	usage *relaymodel.GeminiUsageMetadata,
	imageCount int64,
) model.Usage {
	if usage == nil {
		return geminiImageCountUsage(meta.RequestUsage, imageCount)
	}

	return usage.ToModelUsage()
}

func imageStreamHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	defer resp.Body.Close()

	log := common.GetLogger(c)

	scanner, cleanup := utils.NewStreamScanner(resp.Body, meta.ActualModel)
	defer cleanup()

	usage := meta.RequestUsage
	imageIndex := 0

	var completedB64JSON string

	for scanner.Scan() {
		data := scanner.Bytes()
		if !render.IsValidSSEData(data) {
			continue
		}

		data = render.ExtractSSEData(data)
		if render.IsSSEDone(data) {
			break
		}

		var geminiResponse relaymodel.GeminiChatResponse
		if err := sonic.Unmarshal(data, &geminiResponse); err != nil {
			log.Error("error unmarshalling gemini image stream response: " + err.Error())
			continue
		}

		if geminiResponse.UsageMetadata != nil {
			usage = geminiImageUsageFromResponse(
				meta,
				geminiResponse.UsageMetadata,
				int64(imageIndex),
			)
		}

		for _, candidate := range geminiResponse.Candidates {
			for _, part := range candidate.Content.Parts {
				if part.InlineData == nil ||
					!strings.HasPrefix(part.InlineData.MimeType, "image/") {
					continue
				}

				index := imageIndex
				imageIndex++
				completedB64JSON = part.InlineData.Data

				err := render.ResponsesObjectData(c, relaymodel.ImageStreamEvent{
					Type:              relaymodel.ImageStreamEventPartialImage,
					PartialImageIndex: &index,
					B64Json:           part.InlineData.Data,
				})
				if err != nil {
					log.Warnf("write gemini image stream chunk failed: %v", err)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading gemini image stream: " + err.Error())
	}

	if usage.OutputTokens == 0 && imageIndex > 0 {
		usage = geminiImageCountUsage(meta.RequestUsage, int64(imageIndex))
	}

	imageUsage := usageToImageUsage(usage)

	completed := relaymodel.ImageStreamEvent{
		Type:  relaymodel.ImageStreamEventCompleted,
		Usage: &imageUsage,
	}
	if completedB64JSON != "" {
		completed.B64Json = completedB64JSON
	}

	if err := render.ResponsesObjectData(c, completed); err != nil {
		log.Warnf("write gemini image stream completed failed: %v", err)
	}

	return adaptor.DoResponseResult{Usage: usage}, nil
}

func usageToImageUsage(usage model.Usage) relaymodel.ImageUsage {
	inputTokens := int64(usage.InputTokens)
	imageInputTokens := int64(usage.ImageInputTokens)
	outputTokens := int64(usage.OutputTokens)
	imageOutputTokens := int64(usage.ImageOutputTokens)

	return relaymodel.ImageUsage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  int64(usage.TotalTokens),
		InputTokensDetails: relaymodel.ImageInputTokensDetails{
			TextTokens:  maxInt64(inputTokens-imageInputTokens, 0),
			ImageTokens: imageInputTokens,
		},
		OutputTokensDetails: &relaymodel.ImageOutputTokensDetails{
			ImageTokens: imageOutputTokens,
		},
	}
}

func usageToImageUsagePtr(usage model.Usage) *relaymodel.ImageUsage {
	imageUsage := usageToImageUsage(usage)
	return &imageUsage
}

func geminiImageCountUsage(requestUsage model.Usage, imageCount int64) model.Usage {
	if imageCount == 0 {
		imageCount = int64(requestUsage.OutputTokens)
	}

	return model.Usage{
		InputTokens:       requestUsage.InputTokens,
		ImageInputTokens:  requestUsage.ImageInputTokens,
		OutputTokens:      model.ZeroNullInt64(imageCount),
		ImageOutputTokens: model.ZeroNullInt64(imageCount),
		TotalTokens:       requestUsage.InputTokens + model.ZeroNullInt64(imageCount),
	}
}

func maxInt64(values ...int64) int64 {
	maxValue := int64(0)
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}

	return maxValue
}
