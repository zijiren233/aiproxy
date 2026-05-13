package siliconflow

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/adaptor/registry"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

func init() {
	registry.Register(model.ChannelTypeSiliconflow, &Adaptor{})
}

const baseURL = "https://api.siliconflow.cn/v1"

func (a *Adaptor) DefaultBaseURL() string {
	return baseURL
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "SiliconFlow API\nOpenAI-compatible chat, embeddings, audio, and rerank endpoints\nSupports Gemini-compatible request conversion",
		Models: ModelList,
	}
}

//

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	switch meta.Mode {
	case mode.Embeddings:
		if isVLEmbeddingModel(meta) {
			return openai.ConvertEmbeddingsRequest(meta, req, false, patchVLEmbeddingsInput)
		}

		return a.Adaptor.ConvertRequest(meta, store, req)
	default:
		return a.Adaptor.ConvertRequest(meta, store, req)
	}
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	switch meta.Mode {
	case mode.AudioSpeech:
		if resp.StatusCode != http.StatusOK {
			return adaptor.DoResponseResult{}, ErrorHandler(resp)
		}

		result, err := a.Adaptor.DoResponse(meta, store, c, resp)
		if err != nil {
			return adaptor.DoResponseResult{}, err
		}

		size := c.Writer.Size()
		result.Usage = model.Usage{
			OutputTokens: model.ZeroNullInt64(size),
			TotalTokens:  model.ZeroNullInt64(size),
		}

		return result, nil
	case mode.Rerank:
		if resp.StatusCode != http.StatusOK {
			return adaptor.DoResponseResult{}, ErrorHandler(resp)
		}

		return a.Adaptor.DoResponse(meta, store, c, resp)
	default:
		if !adaptor.IsSuccessfulResponseStatus(meta.Mode, resp.StatusCode) {
			return adaptor.DoResponseResult{}, ErrorHandler(resp)
		}
		return a.Adaptor.DoResponse(meta, store, c, resp)
	}
}
