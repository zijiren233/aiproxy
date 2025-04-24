package jina

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://api.jina.ai/v1"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	switch meta.Mode {
	case mode.Embeddings:
		return ConvertEmbeddingsRequest(meta, req)
	default:
		return a.Adaptor.ConvertRequest(meta, req)
	}
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case mode.Rerank:
		return RerankHandler(meta, c, resp)
	default:
		return a.Adaptor.DoResponse(meta, c, resp)
	}
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}
