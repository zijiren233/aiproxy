package minimax

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct {
	openai.Adaptor
}

var _ adaptor.Adaptor = (*Adaptor)(nil)

const baseURL = "https://api.minimax.chat/v1"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme:  "Chat、Embeddings、TTS(need group id) Support\nGemini support",
		KeyHelp: "api_key|group_id",
		Models:  ModelList,
	}
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	apiKey, _, err := GetAPIKeyAndGroupID(meta.Channel.Key)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)

	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Get groupID for URL construction
	_, groupID, err := GetAPIKeyAndGroupID(meta.Channel.Key)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Determine URL and Method based on mode
	var (
		requestURL string
		urlErr     error
	)

	switch meta.Mode {
	case mode.ChatCompletions, mode.Gemini:
		requestURL, urlErr = url.JoinPath(meta.Channel.BaseURL, "/text/chatcompletion_v2")
		if urlErr != nil {
			return adaptor.ConvertResult{}, urlErr
		}
	case mode.Embeddings:
		requestURL, urlErr = url.JoinPath(meta.Channel.BaseURL, "/embeddings")
		if urlErr != nil {
			return adaptor.ConvertResult{}, urlErr
		}

		requestURL = fmt.Sprintf("%s?GroupId=%s", requestURL, groupID)
	case mode.AudioSpeech:
		requestURL, urlErr = url.JoinPath(meta.Channel.BaseURL, "/t2a_v2")
		if urlErr != nil {
			return adaptor.ConvertResult{}, urlErr
		}

		requestURL = fmt.Sprintf("%s?GroupId=%s", requestURL, groupID)
	default:
		// For other modes, delegate to parent adaptor
		result, err := a.Adaptor.ConvertRequest(meta, store, c, req)
		return result, err
	}

	// Convert request body
	var result adaptor.ConvertResult

	switch meta.Mode {
	case mode.ChatCompletions:
		result, err = openai.ConvertChatCompletionsRequest(meta, req, true)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	case mode.Gemini:
		result, err = openai.ConvertGeminiRequest(meta, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	case mode.AudioSpeech:
		result, err = ConvertTTSRequest(meta, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	default:
		result, err = a.Adaptor.ConvertRequest(meta, store, c, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	}

	// Set URL and Method
	result.Method = http.MethodPost
	result.URL = requestURL

	return result, nil
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		usage, err := TTSHandler(meta, c, resp)
		return adaptor.NewSyncUsage(usage), err
	default:
		if !utils.IsStreamResponse(resp) {
			if err := TryErrorHanlder(resp); err != nil {
				return adaptor.NewSyncUsage(model.Usage{}), err
			}
		}

		return a.Adaptor.DoResponse(meta, store, c, resp)
	}
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}
