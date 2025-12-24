package baiduv2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct{}

var _ adaptor.Adaptor = (*Adaptor)(nil)

const (
	baseURL = "https://qianfan.baidubce.com/v2"
)

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions || m == mode.Rerank
}

// https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Fm2vrveyu
var v2ModelMap = map[string]string{
	"ERNIE-Character-8K":         "ernie-char-8k",
	"ERNIE-Character-Fiction-8K": "ernie-char-fiction-8k",
}

func toV2ModelName(modelName string) string {
	if v2Model, ok := v2ModelMap[modelName]; ok {
		return v2Model
	}
	return strings.ToLower(modelName)
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	token, err := GetBearerToken(context.Background(), meta.Channel.Key)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Set URL and Method based on mode
	var (
		requestURL string
		err        error
	)

	switch meta.Mode {
	case mode.ChatCompletions:
		requestURL, err = url.JoinPath(meta.Channel.BaseURL, "/chat/completions")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	case mode.Rerank:
		requestURL, err = url.JoinPath(meta.Channel.BaseURL, "/rerankers")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	// Convert request body
	var result adaptor.ConvertResult

	switch meta.Mode {
	case mode.ChatCompletions, mode.Rerank:
		actModel := meta.ActualModel

		v2Model := toV2ModelName(actModel)
		if v2Model != actModel {
			meta.ActualModel = v2Model
			defer func() { meta.ActualModel = actModel }()
		}

		result, err = openai.ConvertRequest(meta, store, c, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	// Set URL and Method
	result.Method = http.MethodPost
	result.URL = requestURL

	return result, nil
}

func (a *Adaptor) DoRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (*http.Response, error) {
	return utils.DoRequest(req, meta.RequestTimeout)
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	switch meta.Mode {
	case mode.ChatCompletions, mode.Rerank:
		return openai.DoResponse(meta, store, c, resp)
	default:
		return adaptor.NewSyncUsage(model.Usage{}), relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			nil,
			http.StatusBadRequest,
		)
	}
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		KeyHelp: "ak|sk",
		Models:  ModelList,
	}
}
