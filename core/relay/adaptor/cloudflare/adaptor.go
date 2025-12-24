package cloudflare

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

type Adaptor struct {
	openai.Adaptor
}

var _ adaptor.Adaptor = (*Adaptor)(nil)

const baseURL = "https://api.cloudflare.com"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

// WorkerAI cannot be used across accounts with AIGateWay
// https://developers.cloudflare.com/ai-gateway/providers/workersai/#openai-compatible-endpoints
// https://gateway.ai.cloudflare.com/v1/{account_id}/{gateway_id}/workers-ai
func isAIGateWay(baseURL string) bool {
	return strings.HasPrefix(baseURL, "https://gateway.ai.cloudflare.com") &&
		strings.HasSuffix(baseURL, "/workers-ai")
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Call parent's ConvertRequest
	result, err := a.Adaptor.ConvertRequest(meta, store, c, req)
	if err != nil {
		return result, err
	}

	// Merge GetRequestURL logic
	u := meta.Channel.BaseURL
	isAIGateWay := isAIGateWay(u)

	var urlPrefix string
	if isAIGateWay {
		urlPrefix = u
	} else {
		urlPrefix = fmt.Sprintf("%s/client/v4/accounts/%s/ai", u, meta.Channel.Key)
	}

	var requestURL string

	result.Method = http.MethodPost

	switch meta.Mode {
	case mode.ChatCompletions, mode.Gemini:
		requestURL, err = url.JoinPath(urlPrefix, "/v1/chat/completions")
		if err != nil {
			return result, err
		}
	case mode.Embeddings:
		requestURL, err = url.JoinPath(urlPrefix, "/v1/embeddings")
		if err != nil {
			return result, err
		}
	default:
		if isAIGateWay {
			requestURL, err = url.JoinPath(urlPrefix, meta.ActualModel)
			if err != nil {
				return result, err
			}
		} else {
			requestURL, err = url.JoinPath(urlPrefix, "/run", meta.ActualModel)
			if err != nil {
				return result, err
			}
		}
	}

	result.URL = requestURL

	return result, nil
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Models: ModelList,
	}
}
