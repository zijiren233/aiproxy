package ali

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
	"github.com/labring/aiproxy/relay/relaymode"
	"github.com/labring/aiproxy/relay/utils"
)

// https://help.aliyun.com/zh/dashscope/developer-reference/api-details

type Adaptor struct{}

const baseURL = "https://dashscope.aliyuncs.com"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	u := meta.Channel.BaseURL
	if u == "" {
		u = baseURL
	}
	switch meta.Mode {
	case relaymode.ImagesGenerations:
		return u + "/api/v1/services/aigc/text2image/image-synthesis", nil
	case relaymode.ChatCompletions:
		return u + "/compatible-mode/v1/chat/completions", nil
	case relaymode.Completions:
		return u + "/compatible-mode/v1/completions", nil
	case relaymode.Embeddings:
		return u + "/compatible-mode/v1/embeddings", nil
	case relaymode.AudioSpeech, relaymode.AudioTranscription:
		return u + "/api-ws/v1/inference", nil
	case relaymode.Rerank:
		return u + "/api/v1/services/rerank/text-rerank/text-rerank", nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) SetupRequestHeader(meta *meta.Meta, _ *gin.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)

	// req.Header.Set("X-Dashscope-Plugin", meta.Channel.Config.Plugin)
	return nil
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	switch meta.Mode {
	case relaymode.ImagesGenerations:
		return ConvertImageRequest(meta, req)
	case relaymode.Rerank:
		return ConvertRerankRequest(meta, req)
	case relaymode.ChatCompletions, relaymode.Completions, relaymode.Embeddings:
		return openai.ConvertRequest(meta, req)
	case relaymode.AudioSpeech:
		return ConvertTTSRequest(meta, req)
	case relaymode.AudioTranscription:
		return ConvertSTTRequest(meta, req)
	default:
		return "", nil, nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoRequest(meta *meta.Meta, _ *gin.Context, req *http.Request) (*http.Response, error) {
	switch meta.Mode {
	case relaymode.AudioSpeech:
		return TTSDoRequest(meta, req)
	case relaymode.AudioTranscription:
		return STTDoRequest(meta, req)
	case relaymode.ChatCompletions:
		fallthrough
	default:
		return utils.DoRequest(req)
	}
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *relaymodel.Usage, err *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case relaymode.ImagesGenerations:
		usage, err = ImageHandler(meta, c, resp)
	case relaymode.ChatCompletions, relaymode.Completions, relaymode.Embeddings:
		usage, err = openai.DoResponse(meta, c, resp)
	case relaymode.Rerank:
		usage, err = RerankHandler(meta, c, resp)
	case relaymode.AudioSpeech:
		usage, err = TTSDoResponse(meta, c, resp)
	case relaymode.AudioTranscription:
		usage, err = STTDoResponse(meta, c, resp)
	default:
		return nil, openai.ErrorWrapperWithMessage(fmt.Sprintf("unsupported mode: %s", meta.Mode), "unsupported_mode", http.StatusBadRequest)
	}
	return
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "ali"
}
