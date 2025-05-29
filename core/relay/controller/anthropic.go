package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/utils"
)

func GetAnthropicRequestPrice(_ *gin.Context, mc model.ModelConfig) (model.Price, error) {
	return mc.Price, nil
}

func GetAnthropicRequestUsage(c *gin.Context, _ model.ModelConfig) (model.Usage, error) {
	textRequest, err := utils.UnmarshalAnthropicMessageRequest(c.Request)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		InputTokens: model.ZeroNullInt64(openai.CountTokenMessages(
			textRequest.Messages,
			textRequest.Model,
		)),
	}, nil
}
