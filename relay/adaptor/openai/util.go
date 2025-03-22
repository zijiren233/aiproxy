package openai

import (
	"github.com/labring/aiproxy/middleware"
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
