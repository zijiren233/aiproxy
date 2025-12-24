package streamlake

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://wanqing.streamlakeapi.com/api/gateway/v1/endpoints"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Completions ||
		m == mode.Anthropic
}

func supportClaudeCodeProxy(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "kat") &&
		strings.Contains(strings.ToLower(modelName), "coder")
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	switch {
	case meta.Mode == mode.Anthropic && supportClaudeCodeProxy(meta.OriginModel):
		result, err := anthropic.ConvertRequest(meta, req)
		if err != nil {
			return result, err
		}

		// Set URL and Method for claude-code-proxy
		u := meta.Channel.BaseURL

		fullURL, urlErr := url.JoinPath(u, meta.ActualModel, "/claude-code-proxy/v1/messages")
		if urlErr != nil {
			return adaptor.ConvertResult{}, urlErr
		}

		result.Method = http.MethodPost
		result.URL = fullURL

		return result, nil
	default:
		return a.Adaptor.ConvertRequest(meta, store, c, req)
	}
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	switch {
	case meta.Mode == mode.Anthropic && supportClaudeCodeProxy(meta.OriginModel):
		var (
			usage model.Usage
			err   adaptor.Error
		)

		if utils.IsStreamResponse(resp) {
			usage, err = anthropic.StreamHandler(meta, c, resp)
		} else {
			usage, err = anthropic.Handler(meta, c, resp)
		}

		return adaptor.NewSyncUsage(usage), err
	default:
		return a.Adaptor.DoResponse(meta, store, c, resp)
	}
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Models: ModelList,
	}
}
