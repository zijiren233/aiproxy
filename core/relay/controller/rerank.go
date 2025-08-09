package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func getRerankRequest(c *gin.Context) (*relaymodel.RerankRequest, error) {
	rerankRequest, err := utils.UnmarshalRerankRequest(c.Request)
	if err != nil {
		return nil, err
	}

	if rerankRequest.Model == "" {
		return nil, errors.New("model parameter must be provided")
	}

	if rerankRequest.Query == "" {
		return nil, errors.New("query must not be empty")
	}

	if len(rerankRequest.Documents) == 0 {
		return nil, errors.New("document list must not be empty")
	}

	return rerankRequest, nil
}

func rerankPromptTokens(rerankRequest *relaymodel.RerankRequest) int64 {
	tokens := openai.CountTokenInput(rerankRequest.Query, rerankRequest.Model)
	for _, d := range rerankRequest.Documents {
		tokens += openai.CountTokenInput(d, rerankRequest.Model)
	}

	return tokens
}

func GetRerankRequestUsage(c *gin.Context, _ model.ModelConfig) (model.Usage, error) {
	rerankRequest, err := getRerankRequest(c)
	if err != nil {
		return model.Usage{}, err
	}

	return model.Usage{
		InputTokens: model.ZeroNullInt64(rerankPromptTokens(rerankRequest)),
	}, nil
}
