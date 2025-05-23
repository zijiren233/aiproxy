package aws

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

const (
	ConvertedRequest = "convertedRequest"
)

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	r, err := anthropic.OpenAIConvertRequest(meta, req)
	if err != nil {
		return "", nil, nil, err
	}
	meta.Set("stream", r.Stream)
	meta.Set(ConvertedRequest, r)
	return "", nil, nil, nil
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode) {
	if meta.GetBool("stream") {
		usage, err = StreamHandler(meta, c)
	} else {
		usage, err = Handler(meta, c)
	}
	return
}
