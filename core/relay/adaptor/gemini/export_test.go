package gemini

import (
	"bytes"
	"context"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func BuildMessagePartForTest(
	message relaymodel.MessageContent,
) *relaymodel.GeminiPart {
	return buildMessageParts(message)
}

func ProcessImageTasksForTest(ctx context.Context, imageTasks []*relaymodel.GeminiPart) error {
	return processImageTasks(ctx, imageTasks)
}

func ProcessMediaTasksForTest(
	ctx context.Context,
	mediaType string,
	mediaTasks []*relaymodel.GeminiPart,
) {
	processMediaTasks(ctx, mediaType, mediaTasks)
}

func ResponseChat2OpenAIForTest(
	meta *meta.Meta,
	response *relaymodel.GeminiChatResponse,
) *relaymodel.TextResponse {
	return responseChat2OpenAI(meta, response)
}

func StreamResponseChat2OpenAIForTest(
	meta *meta.Meta,
	response *relaymodel.GeminiChatResponse,
) *relaymodel.ChatCompletionsStreamResponse {
	return streamResponseChat2OpenAI(meta, response)
}

func ConvertVideoRequestForTest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	switch meta.Mode {
	case mode.VideoGenerationsJobs:
		return ConvertVideoGenerationJobRequest(meta, req)
	case mode.Videos:
		return ConvertVideosRequest(meta, req)
	default:
		return ConvertVideoGenerationJobRequest(meta, req)
	}
}

func ConvertRequestWithConfigForTest(
	meta *meta.Meta,
	req *http.Request,
	cfg Config,
) (adaptor.ConvertResult, error) {
	return convertRequest(meta, req, cfg)
}

func ConvertVideoRequestWithConfigForTest(
	meta *meta.Meta,
	req *http.Request,
	cfg Config,
) (adaptor.ConvertResult, error) {
	var request geminiVideoRequest

	var err error
	if meta != nil && meta.Mode == mode.Videos {
		request, err = parseOpenAIVideosRequest(req)
	} else {
		request, err = parseOpenAIVideoGenerationJobRequest(req)
	}

	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	applyVideoPersonGenerationConfig(&request, cfg)

	data, err := sonic.Marshal(request)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type": {"application/json"},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func GeminiImageAspectRatioFromSizeForTest(size string) string {
	return geminiImageAspectRatioFromSize(size)
}

func GeminiVideoURLByIDForTest(operation *relaymodel.GeminiVideoOperation, id string) string {
	return geminiVideoURLByID(operation, id)
}

func GeminiVideoLocalIDForTest(operationName string) string {
	return geminiVideoLocalID(operationName)
}
