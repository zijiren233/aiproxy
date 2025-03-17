package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/common/config"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/utils"
)

func RelayTextHelper(meta *meta.Meta, c *gin.Context) *HandleResult {
	return Handle(meta, c, func() (*PreCheckGroupBalanceReq, error) {
		if !config.GetBillingEnabled() {
			return &PreCheckGroupBalanceReq{}, nil
		}

		inputPrice, outputPrice, cachedPrice, cacheCreationPrice, ok := GetModelPrice(meta.ModelConfig)
		if !ok {
			return nil, fmt.Errorf("model price not found: %s", meta.OriginModel)
		}

		textRequest, err := utils.UnmarshalGeneralOpenAIRequest(c.Request)
		if err != nil {
			return nil, err
		}

		return &PreCheckGroupBalanceReq{
			InputTokens:        openai.GetPromptTokens(meta, textRequest),
			MaxTokens:          textRequest.MaxTokens,
			InputPrice:         inputPrice,
			OutputPrice:        outputPrice,
			CachedPrice:        cachedPrice,
			CacheCreationPrice: cacheCreationPrice,
		}, nil
	})
}
