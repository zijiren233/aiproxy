package doubao

import (
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
	"github.com/labring/aiproxy/core/relay/utils"
)

func getRequestURL(meta *meta.Meta) (method, fullURL string, err error) {
	u := meta.Channel.BaseURL
	switch meta.Mode {
	case mode.ChatCompletions, mode.Anthropic:
		if strings.HasPrefix(meta.ActualModel, "bot-") {
			fullURL, err = url.JoinPath(u, "/api/v3/bots/chat/completions")
			return http.MethodPost, fullURL, err
		}

		fullURL, err = url.JoinPath(u, "/api/v3/chat/completions")

		return http.MethodPost, fullURL, err
	case mode.Embeddings:
		if strings.Contains(meta.ActualModel, "vision") {
			fullURL, err = url.JoinPath(u, "/api/v3/embeddings/multimodal")
			return http.MethodPost, fullURL, err
		}

		fullURL, err = url.JoinPath(u, "/api/v3/embeddings")

		return http.MethodPost, fullURL, err
	default:
		return "", "", fmt.Errorf("unsupported relay mode %d for doubao", meta.Mode)
	}
}

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://ark.cn-beijing.volces.com"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Anthropic ||
		m == mode.Embeddings
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "Bot support\nNetwork search metering support",
		Models: ModelList,
	}
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	var (
		result adaptor.ConvertResult
		err    error
	)

	switch meta.Mode {
	case mode.Embeddings:
		if strings.Contains(meta.ActualModel, "vision") {
			result, err = openai.ConvertEmbeddingsRequest(
				meta,
				req,
				false,
				patchEmbeddingsVisionInput,
			)
		} else {
			result, err = openai.ConvertEmbeddingsRequest(meta, req, true)
		}
	case mode.ChatCompletions:
		result, err = ConvertChatCompletionsRequest(meta, req)
	default:
		result, err = openai.ConvertRequest(meta, store, c, req)
	}

	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Get URL
	method, fullURL, err := getRequestURL(meta)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	result.Method = method
	result.URL = fullURL

	return result, nil
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	var (
		usage model.Usage
		err   adaptor.Error
	)

	switch meta.Mode {
	case mode.ChatCompletions:
		websearchCount := int64(0)
		if utils.IsStreamResponse(resp) {
			usage, err = openai.StreamHandler(meta, c, resp, newHandlerPreHandler(&websearchCount))
		} else {
			usage, err = openai.Handler(meta, c, resp, newHandlerPreHandler(&websearchCount))
		}

		usage.WebSearchCount += model.ZeroNullInt64(websearchCount)
	case mode.Embeddings:
		usage, err = openai.EmbeddingsHandler(
			meta,
			c,
			resp,
			embeddingPreHandler,
		)
	default:
		return openai.DoResponse(meta, store, c, resp)
	}

	return adaptor.NewSyncUsage(usage), err
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}
