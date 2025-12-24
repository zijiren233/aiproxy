package textembeddingsinference

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
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

// text-embeddings-inference adaptor supports rerank and embeddings models deployed by
// https://github.com/huggingface/text-embeddings-inference
type Adaptor struct{}

// base url for text-embeddings-inference, fake
const baseURL = "https://api.text-embeddings.net"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.Rerank || m == mode.Embeddings
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "https://github.com/huggingface/text-embeddings-inference\nEmbeddings、Rerank Support",
		Models: ModelList,
	}
}

// text-embeddings-inference api see
// https://huggingface.github.io/text-embeddings-inference/#/Text%20Embeddings%20Inference/rerank

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
	var (
		result     adaptor.ConvertResult
		err        error
		requestURL string
	)

	switch meta.Mode {
	case mode.Rerank:
		result, err = ConvertRerankRequest(meta, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		requestURL, err = url.JoinPath(meta.Channel.BaseURL, "/rerank")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	case mode.Embeddings:
		result, err = openai.ConvertRequest(meta, store, c, req)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		requestURL, err = url.JoinPath(meta.Channel.BaseURL, "/v1/embeddings")
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	default:
		return adaptor.ConvertResult{}, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	result.Method = http.MethodPost
	result.URL = requestURL

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
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	var (
		usage model.Usage
		err   adaptor.Error
	)

	switch meta.Mode {
	case mode.Rerank:
		usage, err = RerankHandler(meta, c, resp)
	case mode.Embeddings:
		usage, err = EmbeddingsHandler(meta, store, c, resp)
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
