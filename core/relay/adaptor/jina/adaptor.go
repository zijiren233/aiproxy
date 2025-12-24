package jina

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://api.jina.ai/v1"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	switch meta.Mode {
	case mode.Embeddings:
		return ConvertEmbeddingsRequest(meta, req)
	default:
		return a.Adaptor.ConvertRequest(meta, store, c, req)
	}
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	switch meta.Mode {
	case mode.Rerank:
		usage, err := RerankHandler(meta, c, resp)
		if err != nil {
			return nil, err
		}

		return adaptor.NewSyncUsage(usage), nil
	default:
		return a.Adaptor.DoResponse(meta, store, c, resp)
	}
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "https://jina.ai\nEmbeddings、Rerank Support\nGemini support",
		Models: ModelList,
	}
}
