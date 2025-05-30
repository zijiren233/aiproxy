package xai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://api.x.ai/v1"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (usage *model.Usage, err adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHandler(resp)
	}

	return a.Adaptor.DoResponse(meta, c, resp)
}

func (a *Adaptor) GetModelList() []model.ModelConfig {
	return ModelList
}
