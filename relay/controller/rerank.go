package controller

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	relaymodel "github.com/labring/aiproxy/relay/model"
	"github.com/labring/aiproxy/relay/utils"
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

func rerankPromptTokens(rerankRequest *relaymodel.RerankRequest) int {
	return len(rerankRequest.Query) + len(strings.Join(rerankRequest.Documents, ""))
}

func GetRerankRequestPrice(c *gin.Context, mc *model.ModelConfig) (*model.Price, error) {
	return &mc.Price, nil
}

func GetRerankRequestUsage(c *gin.Context, mc *model.ModelConfig) (*model.Usage, error) {
	rerankRequest, err := getRerankRequest(c)
	if err != nil {
		return nil, err
	}
	return &model.Usage{
		InputTokens: rerankPromptTokens(rerankRequest),
	}, nil
}
