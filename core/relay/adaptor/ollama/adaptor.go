package ollama

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct{}

var _ adaptor.Adaptor = (*Adaptor)(nil)

const baseURL = "http://localhost:11434"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.Embeddings || m == mode.ChatCompletions || m == mode.Completions
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)
	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	request *http.Request,
) (adaptor.ConvertResult, error) {
	if request == nil {
		return adaptor.ConvertResult{}, errors.New("request is nil")
	}

	// Construct URL based on mode
	// https://github.com/ollama/ollama/blob/main/docs/api.md
	u := meta.Channel.BaseURL

	var (
		fullURL string
		err     error
	)

	switch meta.Mode {
	case mode.Embeddings:
		fullURL, err = url.JoinPath(u, "/api/embed")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	case mode.ChatCompletions:
		fullURL, err = url.JoinPath(u, "/api/chat")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	case mode.Completions:
		fullURL, err = url.JoinPath(u, "/api/generate")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	// Convert request body
	var result adaptor.ConvertResult
	switch meta.Mode {
	case mode.Embeddings:
		result, err = ConvertEmbeddingRequest(meta, request)
	case mode.ChatCompletions, mode.Completions:
		result, err = ConvertRequest(meta, request)
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Set Method and URL
	result.Method = http.MethodPost
	result.URL = fullURL

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
	_ adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	var (
		usage model.Usage
		err   adaptor.Error
	)

	switch meta.Mode {
	case mode.Embeddings:
		usage, err = EmbeddingHandler(meta, c, resp)
	case mode.ChatCompletions, mode.Completions:
		if utils.IsStreamResponse(resp) {
			usage, err = StreamHandler(meta, c, resp)
		} else {
			usage, err = Handler(meta, c, resp)
		}
	default:
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			"unsupported_mode",
			http.StatusBadRequest,
		)
	}

	if err != nil {
		return nil, err
	}

	return adaptor.NewSyncUsage(usage), nil
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "Chat、Embeddings Support",
		Models: ModelList,
	}
}
