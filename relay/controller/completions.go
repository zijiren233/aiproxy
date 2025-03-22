package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/utils"
)

func GetCompletionsRequestPrice(_ *gin.Context, mc *model.ModelConfig) (model.Price, error) {
	return mc.Price, nil
}

func GetCompletionsRequestUsage(c *gin.Context, _ *model.ModelConfig) (model.Usage, error) {
	textRequest, err := utils.UnmarshalGeneralOpenAIRequest(c.Request)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		InputTokens: openai.CountTokenInput(textRequest.Prompt, textRequest.Model),
	}, nil
}
