package gemini

import (
	"fmt"
	"net/http"

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

const baseURL = "https://generativelanguage.googleapis.com"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Anthropic ||
		m == mode.Embeddings ||
		m == mode.Gemini
}

var v1ModelMap = map[string]struct{}{}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("X-Goog-Api-Key", meta.Channel.Key)
	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Determine action based on mode
	var action string
	switch meta.Mode {
	case mode.Embeddings:
		action = "batchEmbedContents"
	default:
		action = "generateContent"
	}

	if meta.GetBool("stream") ||
		(meta.Mode == mode.Gemini && utils.IsGeminiStreamRequest(c.Request.URL.Path)) {
		action = "streamGenerateContent?alt=sse"
	}

	// Construct URL
	u := meta.Channel.BaseURL
	if u == "" {
		u = baseURL
	}

	version := "v1beta"
	if _, ok := v1ModelMap[meta.ActualModel]; ok {
		version = "v1"
	}

	fullURL := fmt.Sprintf("%s/%s/models/%s:%s", u, version, meta.ActualModel, action)

	// Convert request body
	var (
		result adaptor.ConvertResult
		err    error
	)

	switch meta.Mode {
	case mode.Embeddings:
		result, err = ConvertEmbeddingRequest(meta, req)
	case mode.ChatCompletions:
		result, err = ConvertRequest(meta, req)
	case mode.Anthropic:
		result, err = ConvertClaudeRequest(meta, req)
	case mode.Gemini:
		result, err = NativeConvertRequest(meta, req)
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
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			usage, err = StreamHandler(meta, c, resp)
		} else {
			usage, err = Handler(meta, c, resp)
		}
	case mode.Anthropic:
		if utils.IsStreamResponse(resp) {
			usage, err = ClaudeStreamHandler(meta, c, resp)
		} else {
			usage, err = ClaudeHandler(meta, c, resp)
		}
	case mode.Gemini:
		// For Gemini mode (native format), pass through the response as-is
		if utils.IsStreamResponse(resp) {
			usage, err = NativeStreamHandler(meta, c, resp)
		} else {
			usage, err = NativeHandler(meta, c, resp)
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
		Readme: "https://ai.google.dev\nChat、Embeddings、Image generation Support",
		Models: ModelList,
	}
}
