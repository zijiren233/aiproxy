package aws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/meta"
)

const (
	ConvertedRequest = "convertedRequest"
)

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	r, err := anthropic.OpenAIConvertRequest(meta, req)
	if err != nil {
		return nil, err
	}
	meta.Set("stream", r.Stream)
	meta.Set(ConvertedRequest, r)
	return &adaptor.ConvertRequestResult{
		Method: http.MethodPost,
		Header: nil,
		Body:   nil,
	}, nil
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	_ adaptor.Store,
	c *gin.Context,
) (usage *model.Usage, err adaptor.Error) {
	if meta.GetBool("stream") {
		usage, err = StreamHandler(meta, c)
	} else {
		usage, err = Handler(meta, c)
	}
	return
}
