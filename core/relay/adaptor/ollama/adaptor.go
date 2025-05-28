package ollama

import (
	"errors"
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

const baseURL = "http://localhost:11434"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	// https://github.com/ollama/ollama/blob/main/docs/api.md
	u := meta.Channel.BaseURL
	switch meta.Mode {
	case mode.Embeddings:
		return u + "/api/embed", nil
	case mode.ChatCompletions:
		return u + "/api/chat", nil
	case mode.Completions:
		return u + "/api/generate", nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) SetupRequestHeader(meta *meta.Meta, _ *gin.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)
	return nil
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, request *http.Request) (*adaptor.ConvertRequestResult, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	switch meta.Mode {
	case mode.Embeddings:
		return ConvertEmbeddingRequest(meta, request)
	case mode.ChatCompletions, mode.Completions:
		return ConvertRequest(meta, request)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoRequest(_ *meta.Meta, _ *gin.Context, req *http.Request) (*http.Response, error) {
	return utils.DoRequest(req)
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err adaptor.Error) {
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
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(fmt.Sprintf("unsupported mode: %s", meta.Mode), "unsupported_mode", http.StatusBadRequest)
	}
	return
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}
