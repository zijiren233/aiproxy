package cohere

import (
	"bytes"
	"errors"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct{}

const baseURL = "https://api.cohere.ai"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta, _ adaptor.Store) (string, error) {
	return meta.Channel.BaseURL + "/v1/chat", nil
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)
	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	request, err := utils.UnmarshalGeneralOpenAIRequest(req)
	if err != nil {
		return nil, err
	}
	request.Model = meta.ActualModel
	requestBody := ConvertRequest(request)
	if requestBody == nil {
		return nil, errors.New("request body is nil")
	}
	data, err := sonic.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	return &adaptor.ConvertRequestResult{
		Method: http.MethodPost,
		Header: nil,
		Body:   bytes.NewReader(data),
	}, nil
}

func (a *Adaptor) DoRequest(
	_ *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (*http.Response, error) {
	return utils.DoRequest(req)
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	_ adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (usage *model.Usage, err adaptor.Error) {
	switch meta.Mode {
	case mode.Rerank:
		usage, err = openai.RerankHandler(meta, c, resp)
	default:
		if utils.IsStreamResponse(resp) {
			usage, err = StreamHandler(meta, c, resp)
		} else {
			usage, err = Handler(meta, c, resp)
		}
	}
	return
}

func (a *Adaptor) GetModelList() []model.ModelConfig {
	return ModelList
}
