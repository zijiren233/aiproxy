package openai

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

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct{}

const baseURL = "https://api.openai.com/v1"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Completions ||
		m == mode.Embeddings ||
		m == mode.Moderations ||
		m == mode.ImagesGenerations ||
		m == mode.ImagesEdits ||
		m == mode.AudioSpeech ||
		m == mode.AudioTranscription ||
		m == mode.AudioTranslation ||
		m == mode.Rerank ||
		m == mode.ParsePdf ||
		m == mode.VideoGenerationsJobs ||
		m == mode.VideoGenerationsGetJobs ||
		m == mode.VideoGenerationsContent ||
		m == mode.Anthropic ||
		m == mode.Gemini ||
		m == mode.Responses ||
		m == mode.ResponsesGet ||
		m == mode.ResponsesDelete ||
		m == mode.ResponsesCancel ||
		m == mode.ResponsesInputItems
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
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	return ConvertRequest(meta, store, c, req)
}

func GetRequestURL(meta *meta.Meta) (method, fullURL string, err error) {
	u := meta.Channel.BaseURL

	switch meta.Mode {
	case mode.Responses:
		fullURL, err = url.JoinPath(u, "/responses")
		return http.MethodPost, fullURL, err
	case mode.ResponsesGet:
		fullURL, err = url.JoinPath(u, "/responses", meta.ResponseID)
		return http.MethodGet, fullURL, err
	case mode.ResponsesDelete:
		fullURL, err = url.JoinPath(u, "/responses", meta.ResponseID)
		return http.MethodDelete, fullURL, err
	case mode.ResponsesCancel:
		fullURL, err = url.JoinPath(u, "/responses", meta.ResponseID, "cancel")
		return http.MethodPost, fullURL, err
	case mode.ResponsesInputItems:
		fullURL, err = url.JoinPath(u, "/responses", meta.ResponseID, "input_items")
		return http.MethodGet, fullURL, err
	case mode.ChatCompletions, mode.Anthropic, mode.Gemini:
		// Check if model requires Responses API
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			fullURL, err = url.JoinPath(u, "/responses")
			return http.MethodPost, fullURL, err
		}

		fullURL, err = url.JoinPath(u, "/chat/completions")

		return http.MethodPost, fullURL, err
	case mode.Completions:
		fullURL, err = url.JoinPath(u, "/completions")
		return http.MethodPost, fullURL, err
	case mode.Embeddings:
		fullURL, err = url.JoinPath(u, "/embeddings")
		return http.MethodPost, fullURL, err
	case mode.Moderations:
		fullURL, err = url.JoinPath(u, "/moderations")
		return http.MethodPost, fullURL, err
	case mode.ImagesGenerations:
		fullURL, err = url.JoinPath(u, "/images/generations")
		return http.MethodPost, fullURL, err
	case mode.ImagesEdits:
		fullURL, err = url.JoinPath(u, "/images/edits")
		return http.MethodPost, fullURL, err
	case mode.AudioSpeech:
		fullURL, err = url.JoinPath(u, "/audio/speech")
		return http.MethodPost, fullURL, err
	case mode.AudioTranscription:
		fullURL, err = url.JoinPath(u, "/audio/transcriptions")
		return http.MethodPost, fullURL, err
	case mode.AudioTranslation:
		fullURL, err = url.JoinPath(u, "/audio/translations")
		return http.MethodPost, fullURL, err
	case mode.Rerank:
		fullURL, err = url.JoinPath(u, "/rerank")
		return http.MethodPost, fullURL, err
	case mode.VideoGenerationsJobs:
		fullURL, err = url.JoinPath(u, "/video/generations/jobs")
		return http.MethodPost, fullURL, err
	case mode.VideoGenerationsGetJobs:
		fullURL, err = url.JoinPath(u, "/video/generations/jobs", meta.JobID)
		return http.MethodGet, fullURL, err
	case mode.VideoGenerationsContent:
		fullURL, err = url.JoinPath(u, "/video/generations", meta.GenerationID, "/content/video")
		return http.MethodGet, fullURL, err
	default:
		return "", "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	if req == nil {
		return adaptor.ConvertResult{}, errors.New("request is nil")
	}

	switch meta.Mode {
	case mode.Responses:
		return ConvertResponseRequest(meta, req)
	case mode.ResponsesGet:
		return ConvertResponsesGetRequest(meta, req)
	case mode.ResponsesDelete:
		return ConvertResponsesDeleteRequest(meta, req)
	case mode.ResponsesCancel:
		return ConvertResponsesCancelRequest(meta, req)
	case mode.ResponsesInputItems:
		return ConvertResponsesInputItemsRequest(meta, req)
	case mode.Moderations:
		return ConvertModerationsRequest(meta, req)
	case mode.Embeddings:
		return ConvertEmbeddingsRequest(meta, req, false)
	case mode.Completions:
		return ConvertCompletionsRequest(meta, req)
	case mode.ChatCompletions:
		// Check if model requires Responses API conversion
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			return ConvertChatCompletionToResponsesRequest(meta, req)
		}
		return ConvertChatCompletionsRequest(meta, req, false)
	case mode.Anthropic:
		// Check if model requires Responses API conversion
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			return ConvertClaudeToResponsesRequest(meta, req)
		}
		return ConvertClaudeRequest(meta, req)
	case mode.ImagesGenerations:
		return ConvertImagesRequest(meta, req)
	case mode.ImagesEdits:
		return ConvertImagesEditsRequest(meta, req)
	case mode.AudioTranscription, mode.AudioTranslation:
		return ConvertSTTRequest(meta, req)
	case mode.AudioSpeech:
		return ConvertTTSRequest(meta, req, "")
	case mode.Rerank:
		return ConvertRerankRequest(meta, req)
	case mode.VideoGenerationsJobs:
		return ConvertVideoRequest(meta, req)
	case mode.VideoGenerationsGetJobs:
		return ConvertVideoGetJobsRequest(meta, req)
	case mode.VideoGenerationsContent:
		return ConvertVideoGetJobsContentRequest(meta, req)
	case mode.Gemini:
		// Check if model requires Responses API conversion
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			return ConvertGeminiToResponsesRequest(meta, req)
		}
		return ConvertGeminiRequest(meta, req)
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func DoResponse(
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
	case mode.Responses:
		if utils.IsStreamResponse(resp) {
			usage, err = ResponseStreamHandler(meta, store, c, resp)
		} else {
			usage, err = ResponseHandler(meta, store, c, resp)
		}
	case mode.ResponsesGet:
		usage, err = GetResponseHandler(meta, c, resp)
	case mode.ResponsesDelete:
		usage, err = DeleteResponseHandler(meta, c, resp)
	case mode.ResponsesCancel:
		usage, err = CancelResponseHandler(meta, c, resp)
	case mode.ResponsesInputItems:
		usage, err = GetInputItemsHandler(meta, c, resp)
	case mode.ImagesGenerations, mode.ImagesEdits:
		usage, err = ImagesHandler(meta, c, resp)
	case mode.AudioTranscription, mode.AudioTranslation:
		usage, err = STTHandler(meta, c, resp)
	case mode.AudioSpeech:
		usage, err = TTSHandler(meta, c, resp)
	case mode.Rerank:
		usage, err = RerankHandler(meta, c, resp)
	case mode.Moderations:
		usage, err = ModerationsHandler(meta, c, resp)
	case mode.Embeddings:
		usage, err = EmbeddingsHandler(meta, c, resp, nil)
	case mode.Completions, mode.ChatCompletions:
		// Check if model required Responses API conversion
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			// Convert Responses API response back to ChatCompletion format
			if utils.IsStreamResponse(resp) {
				usage, err = ConvertResponsesToChatCompletionStreamResponse(meta, c, resp)
			} else {
				usage, err = ConvertResponsesToChatCompletionResponse(meta, c, resp)
			}
		} else {
			if utils.IsStreamResponse(resp) {
				usage, err = StreamHandler(meta, c, resp, nil)
			} else {
				usage, err = Handler(meta, c, resp, nil)
			}
		}
	case mode.Anthropic:
		// Check if model required Responses API conversion
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			// Convert Responses API response back to Claude format
			if utils.IsStreamResponse(resp) {
				usage, err = ConvertResponsesToClaudeStreamResponse(meta, c, resp)
			} else {
				usage, err = ConvertResponsesToClaudeResponse(meta, c, resp)
			}
		} else {
			if utils.IsStreamResponse(resp) {
				usage, err = ClaudeStreamHandler(meta, c, resp)
			} else {
				usage, err = ClaudeHandler(meta, c, resp)
			}
		}
	case mode.VideoGenerationsJobs:
		return VideoHandlerWithUsageResult(meta, store, c, resp)
	case mode.VideoGenerationsGetJobs:
		usage, err = VideoGetJobsHandler(meta, store, c, resp)
	case mode.VideoGenerationsContent:
		usage, err = VideoGetJobsContentHandler(meta, store, c, resp)
	case mode.Gemini:
		// Check if model required Responses API conversion
		if IsResponsesOnlyModel(&meta.ModelConfig, meta.ActualModel) {
			// Convert Responses API response back to Gemini format
			if utils.IsStreamResponse(resp) {
				usage, err = ConvertResponsesToGeminiStreamResponse(meta, c, resp)
			} else {
				usage, err = ConvertResponsesToGeminiResponse(meta, c, resp)
			}
		} else {
			if utils.IsStreamResponse(resp) {
				usage, err = GeminiStreamHandler(meta, c, resp)
			} else {
				usage, err = GeminiHandler(meta, c, resp)
			}
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

const MetaResponseFormat = "response_format"

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
	return DoResponse(meta, store, c, resp)
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "OpenAI compatibility\nAnthropic conversation",
		Models: ModelList,
	}
}
