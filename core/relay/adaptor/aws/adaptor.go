package aws

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/aws/utils"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type Adaptor struct{}

func (a *Adaptor) GetBaseURL() string {
	return ""
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (*adaptor.ConvertRequestResult, error) {
	adaptor := GetAdaptor(meta.ActualModel)
	if adaptor == nil {
		return nil, errors.New("adaptor not found")
	}
	meta.Set("awsAdapter", adaptor)
	return adaptor.ConvertRequest(meta, req)
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, _ *http.Response) (usage *model.Usage, err adaptor.Error) {
	adaptor, ok := meta.Get("awsAdapter")
	if !ok {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage("awsAdapter not found", nil, http.StatusInternalServerError)
	}
	return adaptor.(utils.AwsAdapter).DoResponse(meta, c)
}

func (a *Adaptor) GetModelList() (models []model.ModelConfig) {
	models = make([]model.ModelConfig, 0, len(adaptors))
	for _, model := range adaptors {
		models = append(models, model.config)
	}
	return
}

func (a *Adaptor) GetRequestURL(_ *meta.Meta) (string, error) {
	return "", nil
}

func (a *Adaptor) SetupRequestHeader(_ *meta.Meta, _ *gin.Context, _ *http.Request) error {
	return nil
}

func (a *Adaptor) DoRequest(_ *meta.Meta, _ *gin.Context, _ *http.Request) (*http.Response, error) {
	return nil, nil
}
