package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/utils"
)

func GetChatRequestPrice(_ *gin.Context, mc *model.ModelConfig) (*model.Price, error) {
	return &mc.Price, nil
}

func GetChatRequestUsage(c *gin.Context, _ *model.ModelConfig) (*model.Usage, error) {
	textRequest, err := utils.UnmarshalGeneralOpenAIRequest(c.Request)
	if err != nil {
		return nil, err
	}

	return &model.Usage{
		InputTokens: openai.CountTokenMessages(textRequest.Messages, textRequest.Model),
	}, nil
}
