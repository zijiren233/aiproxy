package ali

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

// https://help.aliyun.com/zh/dashscope/developer-reference/api-details

type Adaptor struct{}

var _ adaptor.Adaptor = (*Adaptor)(nil)

const baseURL = "https://dashscope.aliyuncs.com"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Completions ||
		m == mode.Embeddings ||
		m == mode.ImagesGenerations ||
		m == mode.Rerank ||
		m == mode.AudioSpeech ||
		m == mode.AudioTranscription ||
		m == mode.AudioTranslation ||
		m == mode.Anthropic ||
		m == mode.Gemini
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)

	// req.Header.Set("X-Dashscope-Plugin", meta.Channel.Config.Plugin)
	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Construct URL based on mode
	u := meta.Channel.BaseURL

	var (
		fullURL string
		err     error
	)

	switch meta.Mode {
	case mode.ImagesGenerations:
		fullURL, err = url.JoinPath(u, "/api/v1/services/aigc/text2image/image-synthesis")
	case mode.ChatCompletions:
		fullURL, err = url.JoinPath(u, "/compatible-mode/v1/chat/completions")
	case mode.Completions:
		fullURL, err = url.JoinPath(u, "/compatible-mode/v1/completions")
	case mode.Embeddings:
		fullURL, err = url.JoinPath(u, "/compatible-mode/v1/embeddings")
	case mode.AudioSpeech, mode.AudioTranscription:
		fullURL, err = url.JoinPath(u, "/api-ws/v1/inference")
	case mode.Rerank:
		fullURL, err = url.JoinPath(u, "/api/v1/services/rerank/text-rerank/text-rerank")
	case mode.Anthropic:
		fullURL, err = url.JoinPath(u, "/api/v2/apps/claude-code-proxy/v1/messages")
	case mode.Gemini:
		fullURL, err = url.JoinPath(u, "/compatible-mode/v1/chat/completions")
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Convert request body
	var result adaptor.ConvertResult
	switch meta.Mode {
	case mode.ImagesGenerations:
		result, err = ConvertImageRequest(meta, req)
	case mode.Rerank:
		result, err = ConvertRerankRequest(meta, req)
	case mode.ChatCompletions:
		result, err = ConvertChatCompletionsRequest(meta, store, req)
	case mode.Completions:
		result, err = ConvertCompletionsRequest(meta, store, req)
	case mode.Embeddings:
		result, err = openai.ConvertRequest(meta, store, nil, req)
	case mode.AudioSpeech:
		result, err = ConvertTTSRequest(meta, req)
	case mode.AudioTranscription:
		result, err = ConvertSTTRequest(meta, req)
	case mode.Anthropic:
		result, err = anthropic.ConvertRequest(meta, req)
	case mode.Gemini:
		result, err = openai.ConvertGeminiRequest(meta, req)
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
	switch meta.Mode {
	case mode.AudioSpeech:
		return TTSDoRequest(meta, req)
	case mode.AudioTranscription:
		return STTDoRequest(meta, req)
	default:
		return utils.DoRequest(req, meta.RequestTimeout)
	}
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
	case mode.ImagesGenerations:
		usage, err = ImageHandler(meta, c, resp)
	case mode.Embeddings:
		usage, err = EmbeddingsHandler(meta, store, c, resp)
	case mode.Completions, mode.ChatCompletions:
		usage, err = ChatHandler(meta, store, c, resp)
	case mode.Rerank:
		usage, err = RerankHandler(meta, c, resp)
	case mode.AudioSpeech:
		usage, err = TTSDoResponse(meta, c, resp)
	case mode.AudioTranscription:
		usage, err = STTDoResponse(meta, c, resp)
	case mode.Anthropic:
		if utils.IsStreamResponse(resp) {
			usage, err = anthropic.StreamHandler(meta, c, resp)
		} else {
			usage, err = anthropic.Handler(meta, c, resp)
		}
	case mode.Gemini:
		if utils.IsStreamResponse(resp) {
			usage, err = openai.GeminiStreamHandler(meta, c, resp)
		} else {
			usage, err = openai.GeminiHandler(meta, c, resp)
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
		Readme: "OpenAI compatibility\nNetwork search metering support\nRerank support: https://help.aliyun.com/zh/model-studio/text-rerank-api\nSTT support: https://help.aliyun.com/zh/model-studio/sambert-speech-synthesis/\nAnthropic support: /api/v2/apps/claude-code-proxy\nGemini support",
		Models: ModelList,
	}
}
