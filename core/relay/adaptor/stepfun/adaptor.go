package stepfun

import (
	"net/http"

	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://api.stepfun.com/v1"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		return openai.ConvertTTSRequest(meta, req, "cixingnansheng")
	default:
		return a.Adaptor.ConvertRequest(meta, store, req)
	}
}

func (a *Adaptor) GetModelList() []model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}
