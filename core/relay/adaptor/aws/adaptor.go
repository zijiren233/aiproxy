package aws

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/aws/utils"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type Adaptor struct{}

var _ adaptor.Adaptor = (*Adaptor)(nil)

func (a *Adaptor) DefaultBaseURL() string {
	return ""
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Completions ||
		m == mode.Anthropic ||
		m == mode.Gemini
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	aa := GetAdaptor(meta.ActualModel)
	if aa == nil {
		return adaptor.ConvertResult{}, errors.New("adaptor not found")
	}

	meta.Set("awsAdapter", aa)

	result, err := aa.ConvertRequest(meta, store, req)
	if err != nil {
		return result, err
	}

	result.Method = http.MethodPost
	result.URL = ""

	return result, nil
}

func (a *Adaptor) DoRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (*http.Response, error) {
	awsAdaptor, ok := meta.Get("awsAdapter")
	if !ok {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			"awsAdapter not found",
			nil,
			http.StatusInternalServerError,
		)
	}

	v, ok := awsAdaptor.(utils.AwsAdapter)
	if !ok {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("aws adapter type error: %T, %v", v, v),
			nil,
			http.StatusInternalServerError,
		)
	}

	return v.DoRequest(meta, store, c, req)
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	_ *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	awsAdaptor, ok := meta.Get("awsAdapter")
	if !ok {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			"awsAdapter not found",
			nil,
			http.StatusInternalServerError,
		)
	}

	v, ok := awsAdaptor.(utils.AwsAdapter)
	if !ok {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("aws adapter type error: %T, %v", v, v),
			nil,
			http.StatusInternalServerError,
		)
	}

	usage, err := v.DoResponse(meta, store, c)
	if err != nil {
		return nil, err
	}

	return adaptor.NewSyncUsage(usage), nil
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	models := make([]model.ModelConfig, 0, len(adaptors))
	for _, model := range adaptors {
		models = append(models, model.config)
	}

	return adaptor.Metadata{
		Readme:  "Gemini support",
		Models:  models,
		KeyHelp: "region|ak|sk or region|apikey",
	}
}

func (a *Adaptor) SetupRequestHeader(
	_ *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	_ *http.Request,
) error {
	return nil
}
