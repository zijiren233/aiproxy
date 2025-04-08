package aws

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/relay/adaptor/aws/utils"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

const (
	ConvertedRequest = "convertedRequest"
)

var _ utils.AwsAdapter = new(Adaptor)

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	r, err := anthropic.ConvertRequest(meta, req)
	if err != nil {
		return "", nil, nil, err
	}
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
