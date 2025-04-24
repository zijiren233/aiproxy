package ali

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
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
	case mode.ImagesGenerations:
		return u + "/api/v1/services/aigc/text2image/image-synthesis", nil
	case mode.ChatCompletions:
		return u + "/compatible-mode/v1/chat/completions", nil
	case mode.Completions:
		return u + "/compatible-mode/v1/completions", nil
	case mode.Embeddings:
		return u + "/compatible-mode/v1/embeddings", nil
	case mode.AudioSpeech, mode.AudioTranscription:
		return u + "/api-ws/v1/inference", nil
	case mode.Rerank:
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
	case mode.ImagesGenerations:
		return ConvertImageRequest(meta, req)
	case mode.Rerank:
		return ConvertRerankRequest(meta, req)
	case mode.ChatCompletions, mode.Completions, mode.Embeddings:
		return openai.ConvertRequest(meta, req)
	case mode.AudioSpeech:
		return ConvertTTSRequest(meta, req)
	case mode.AudioTranscription:
		return ConvertSTTRequest(meta, req)
	default:
		return "", nil, nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoRequest(meta *meta.Meta, _ *gin.Context, req *http.Request) (*http.Response, error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		return TTSDoRequest(meta, req)
	case mode.AudioTranscription:
		return STTDoRequest(meta, req)
	case mode.ChatCompletions:
		fallthrough
	default:
		return utils.DoRequest(req)
	}
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case mode.ImagesGenerations:
		return ImageHandler(meta, c, resp)
	case mode.Embeddings, mode.Completions:
		return openai.DoResponse(meta, c, resp)
	case mode.ChatCompletions:
		reqBody, err := common.GetRequestBody(c.Request)
		if err != nil {
			return nil, openai.ErrorWrapperWithMessage(fmt.Sprintf("get request body failed: %s", err), "get_request_body_failed", http.StatusInternalServerError)
		}
		enableSearch, err := getEnableSearch(reqBody)
		if err != nil {
			return nil, openai.ErrorWrapperWithMessage(fmt.Sprintf("get enable_search failed: %s", err), "get_enable_search_failed", http.StatusInternalServerError)
		}
		u, e := openai.DoResponse(meta, c, resp)
		if e != nil {
			return nil, e
		}
		if enableSearch {
			u.WebSearchCount++
		}
		return u, nil
	case mode.Rerank:
		return RerankHandler(meta, c, resp)
	case mode.AudioSpeech:
		return TTSDoResponse(meta, c, resp)
	case mode.AudioTranscription:
		return STTDoResponse(meta, c, resp)
	default:
		return nil, openai.ErrorWrapperWithMessage(fmt.Sprintf("unsupported mode: %s", meta.Mode), "unsupported_mode", http.StatusBadRequest)
	}
}

func getEnableSearch(reqBody []byte) (bool, error) {
	searchNode, err := sonic.Get(reqBody, "enable_search")
	if err != nil {
		if errors.Is(err, ast.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("get enable_search failed: %w", err)
	}
	enableSearch, err := searchNode.Bool()
	if err != nil {
		if errors.Is(err, ast.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("get enable_search failed: %w", err)
	}
	return enableSearch, nil
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}
