package gemini

import (
	"context"

	"github.com/labring/aiproxy/core/relay/meta"
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
