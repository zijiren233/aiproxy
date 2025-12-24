package zhipucoding

import (
	"net/http"
	"net/url"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/adaptor/zhipu"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://open.bigmodel.cn"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Completions ||
		m == mode.Anthropic
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	u := meta.Channel.BaseURL

	switch meta.Mode {
	case mode.Anthropic:
		result, err := anthropic.ConvertRequest(meta, req, func(node *ast.Node) error {
			if !node.Get("max_tokens").Exists() {
				_, err := node.Set("max_tokens", ast.NewNumber("4096"))
				return err
			}

			return nil
		})
		if err != nil {
			return result, err
		}

		// Set URL and Method for Anthropic
		fullURL, urlErr := url.JoinPath(u, "/api/anthropic/v1/messages")
		if urlErr != nil {
			return adaptor.ConvertResult{}, urlErr
		}

		result.Method = http.MethodPost
		result.URL = fullURL

		return result, nil
	default:
		// Temporarily modify BaseURL for other modes
		meta.Channel.BaseURL += "/api/coding/paas/v4"
		defer func() {
			meta.Channel.BaseURL = u
		}()

		return a.Adaptor.ConvertRequest(meta, store, c, req)
	}
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	switch meta.Mode {
	case mode.Anthropic:
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
		Models: zhipu.ModelList,
	}
}
