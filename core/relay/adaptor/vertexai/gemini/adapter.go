package vertexai

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/gemini"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, request *http.Request) (string, http.Header, io.Reader, error) {
	return gemini.ConvertRequest(meta, request)
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case mode.Embeddings:
		usage, err = gemini.EmbeddingHandler(meta, c, resp)
	default:
		if utils.IsStreamResponse(resp) {
			usage, err = gemini.StreamHandler(meta, c, resp)
		} else {
			usage, err = gemini.Handler(meta, c, resp)
		}
	}
	return
}
