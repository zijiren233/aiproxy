package zhipu

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://open.bigmodel.cn/api/paas/v4"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	var (
		usage model.Usage
		err   adaptor.Error
	)

	switch meta.Mode {
	case mode.Embeddings:
		usage, err = EmbeddingsHandler(c, resp)
	default:
		return openai.DoResponse(meta, store, c, resp)
	}

	if err != nil {
		return nil, err
	}

	return adaptor.NewSyncUsage(usage), nil
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "Gemini support",
		Models: ModelList,
	}
}
