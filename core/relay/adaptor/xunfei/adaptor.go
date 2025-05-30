package xunfei

import (
	"net/http"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
)

type Adaptor struct {
	openai.Adaptor
}

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

const baseURL = "https://spark-api-open.xf-yun.com/v1"

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	domain := getXunfeiDomain(meta.ActualModel)
	model := meta.ActualModel
	meta.ActualModel = domain
	defer func() {
		meta.ActualModel = model
	}()
	return a.Adaptor.ConvertRequest(meta, req)
}

func (a *Adaptor) GetModelList() []model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}
