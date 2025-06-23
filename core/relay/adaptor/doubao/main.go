package doubao

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
)

func GetRequestURL(meta *meta.Meta) (adaptor.RequestURL, error) {
	u := meta.Channel.BaseURL
	switch meta.Mode {
	case mode.ChatCompletions:
		if strings.HasPrefix(meta.ActualModel, "bot-") {
			return adaptor.RequestURL{
				Method: http.MethodPost,
				URL:    u + "/api/v3/bots/chat/completions",
			}, nil
		}
		return adaptor.RequestURL{
			Method: http.MethodPost,
			URL:    u + "/api/v3/chat/completions",
		}, nil
	case mode.Embeddings:
		if strings.Contains(meta.ActualModel, "vision") {
			return adaptor.RequestURL{
				Method: http.MethodPost,
				URL:    u + "/api/v3/embeddings/multimodal",
			}, nil
		}
		return adaptor.RequestURL{
			Method: http.MethodPost,
			URL:    u + "/api/v3/embeddings",
		}, nil
	default:
		return adaptor.RequestURL{}, fmt.Errorf("unsupported relay mode %d for doubao", meta.Mode)
	}
}

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://ark.cn-beijing.volces.com"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Features: []string{
			"Bot support",
			"Network search metering support",
		},
		Models: ModelList,
	}
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta, _ adaptor.Store) (adaptor.RequestURL, error) {
	return GetRequestURL(meta)
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	switch meta.Mode {
	case mode.Embeddings:
		if strings.Contains(meta.ActualModel, "vision") {
			return ConvertEmbeddingsVisionRequest(meta, store, req)
		} else {
			return openai.ConvertRequest(meta, store, req)
		}
	case mode.ChatCompletions:
		return ConvertChatCompletionsRequest(meta, req)
	}
	return openai.ConvertRequest(meta, store, req)
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (usage model.Usage, err adaptor.Error) {
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
	return usage, err
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}
