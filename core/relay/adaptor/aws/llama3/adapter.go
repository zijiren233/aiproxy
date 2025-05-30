package aws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relayutils "github.com/labring/aiproxy/core/relay/utils"
)

const (
	ConvertedRequest = "convertedRequest"
)

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	request, err := relayutils.UnmarshalGeneralOpenAIRequest(req)
	if err != nil {
		return nil, err
	}
	request.Model = meta.ActualModel
	meta.Set("stream", request.Stream)
	llamaReq := ConvertRequest(request)
	meta.Set(ConvertedRequest, llamaReq)
	return &adaptor.ConvertRequestResult{
		Method: http.MethodPost,
		Header: nil,
		Body:   nil,
	}, nil
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	c *gin.Context,
) (usage *model.Usage, err adaptor.Error) {
	if meta.GetBool("stream") {
		usage, err = StreamHandler(meta, c)
	} else {
		usage, err = Handler(meta, c)
	}
	return
}
