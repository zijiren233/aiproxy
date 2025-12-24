package doubaoaudio

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func getRequestURL(meta *meta.Meta) (method, fullURL string, err error) {
	u := meta.Channel.BaseURL
	switch meta.Mode {
	case mode.AudioSpeech:
		fullURL, err = url.JoinPath(u, "/api/v1/tts/ws_binary")
		return http.MethodPost, fullURL, err
	default:
		return "", "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

type Adaptor struct{}

const baseURL = "https://openspeech.bytedance.com"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.AudioSpeech
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme:  "https://www.volcengine.com/docs/6561/1257543\nTTS support",
		KeyHelp: "app_id|app_token",
		Models:  ModelList,
	}
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	var (
		result adaptor.ConvertResult
		err    error
	)

	switch meta.Mode {
	case mode.AudioSpeech:
		result, err = ConvertTTSRequest(meta, req)
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
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

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	switch meta.Mode {
	case mode.AudioSpeech:
		_, token, err := getAppIDAndToken(meta.Channel.Key)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", "Bearer;"+token)

		return nil
	default:
		return fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
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
	default:
		return nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	_ adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		usage, err := TTSDoResponse(meta, c, resp)
		return adaptor.NewSyncUsage(usage), err
	default:
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			nil,
			http.StatusBadRequest,
		)
	}
}
