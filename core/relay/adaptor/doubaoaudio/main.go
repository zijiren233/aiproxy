package doubaoaudio

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func GetRequestURL(meta *meta.Meta) (string, error) {
	u := meta.Channel.BaseURL
	switch meta.Mode {
	case mode.AudioSpeech:
		return u + "/api/v1/tts/ws_binary", nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

type Adaptor struct{}

const baseURL = "https://openspeech.bytedance.com"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	return GetRequestURL(meta)
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (*adaptor.ConvertRequestResult, error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		return ConvertTTSRequest(meta, req)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) SetupRequestHeader(meta *meta.Meta, _ *gin.Context, req *http.Request) error {
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

func (a *Adaptor) DoRequest(meta *meta.Meta, _ *gin.Context, req *http.Request) (*http.Response, error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		return TTSDoRequest(meta, req)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, adaptor.Error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		return TTSDoResponse(meta, c, resp)
	default:
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			nil,
			http.StatusBadRequest,
		)
	}
}
