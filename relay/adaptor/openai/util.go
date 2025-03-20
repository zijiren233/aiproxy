package openai

import (
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/mode"
	model "github.com/labring/aiproxy/relay/model"
)

func ErrorWrapper(err error, code any, statusCode int) *model.ErrorWithStatusCode {
	return &model.ErrorWithStatusCode{
		Error: model.Error{
			Message: err.Error(),
			Type:    middleware.ErrorTypeAIPROXY,
			Code:    code,
		},
		StatusCode: statusCode,
	}
}

func ErrorWrapperWithMessage(message string, code any, statusCode int) *model.ErrorWithStatusCode {
	return &model.ErrorWithStatusCode{
		Error: model.Error{
			Message: message,
			Type:    middleware.ErrorTypeAIPROXY,
			Code:    code,
		},
		StatusCode: statusCode,
	}
}

func GetPromptTokens(meta *meta.Meta, textRequest *model.GeneralOpenAIRequest) int {
	switch meta.Mode {
	case mode.ChatCompletions:
		return CountTokenMessages(textRequest.Messages, textRequest.Model)
	case mode.Completions:
		return CountTokenInput(textRequest.Prompt, textRequest.Model)
	case mode.Moderations:
		return CountTokenInput(textRequest.Input, textRequest.Model)
	}
	return 0
}
