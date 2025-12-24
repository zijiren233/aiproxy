package baidu

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct{}

var _ adaptor.Adaptor = (*Adaptor)(nil)

const (
	baseURL = "https://aip.baidubce.com"
)

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions ||
		m == mode.Embeddings ||
		m == mode.Rerank ||
		m == mode.ImagesGenerations
}

// Get model-specific endpoint using map
var modelEndpointMap = map[string]string{
	"ERNIE-4.0-8K":         "completions_pro",
	"ERNIE-4.0":            "completions_pro",
	"ERNIE-Bot-4":          "completions_pro",
	"ERNIE-Bot":            "completions",
	"ERNIE-Bot-turbo":      "eb-instant",
	"ERNIE-Speed":          "ernie_speed",
	"ERNIE-3.5-8K":         "completions",
	"ERNIE-Bot-8K":         "ernie_bot_8k",
	"ERNIE-Speed-8K":       "ernie_speed",
	"ERNIE-Lite-8K-0922":   "eb-instant",
	"ERNIE-Lite-8K-0308":   "ernie-lite-8k",
	"BLOOMZ-7B":            "bloomz_7b1",
	"bge-large-zh":         "bge_large_zh",
	"bge-large-en":         "bge_large_en",
	"tao-8k":               "tao_8k",
	"bce-reranker-base_v1": "bce_reranker_base",
	"Stable-Diffusion-XL":  "sd_xl",
	"Fuyu-8B":              "fuyu_8b",
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)

	accessToken, err := GetAccessToken(context.Background(), meta.Channel.Key)
	if err != nil {
		return err
	}

	req.URL.RawQuery = "access_token=" + accessToken

	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	// Get API path suffix based on mode
	var pathSuffix string
	switch meta.Mode {
	case mode.ChatCompletions:
		pathSuffix = "chat"
	case mode.Embeddings:
		pathSuffix = "embeddings"
	case mode.Rerank:
		pathSuffix = "reranker"
	case mode.ImagesGenerations:
		pathSuffix = "text2image"
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	modelEndpoint, ok := modelEndpointMap[meta.ActualModel]
	if !ok {
		modelEndpoint = strings.ToLower(meta.ActualModel)
	}

	// Construct full URL
	fullURL, err := url.JoinPath(
		meta.Channel.BaseURL,
		"/rpc/2.0/ai_custom/v1/wenxinworkshop",
		pathSuffix,
		modelEndpoint,
	)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Convert request body
	var result adaptor.ConvertResult
	switch meta.Mode {
	case mode.Embeddings:
		result, err = openai.ConvertEmbeddingsRequest(meta, req, true)
	case mode.Rerank:
		result, err = openai.ConvertRequest(meta, store, nil, req)
	case mode.ImagesGenerations:
		result, err = openai.ConvertRequest(meta, store, nil, req)
	case mode.ChatCompletions:
		result, err = ConvertRequest(meta, req)
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
		usage, err = EmbeddingsHandler(meta, c, resp)
	case mode.Rerank:
		usage, err = RerankHandler(meta, c, resp)
	case mode.ImagesGenerations:
		usage, err = ImageHandler(meta, c, resp)
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			usage, err = StreamHandler(meta, c, resp)
		} else {
			usage, err = Handler(meta, c, resp)
		}
	default:
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			nil,
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
		KeyHelp: "client_id|client_secret",
		Models:  ModelList,
	}
}
