package vertexai

import (
	"bytes"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
	"github.com/labring/aiproxy/relay/relaymode"
	"github.com/labring/aiproxy/relay/utils"
	"github.com/pkg/errors"
)

var ModelList = []*model.ModelConfig{
	{
		Model: "claude-3-haiku@20240307",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-sonnet@20240229",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-opus@20240229",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-5-sonnet@20240620",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-5-sonnet-v2@20241022",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-5-haiku@20241022",
		Type:  relaymode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
}

const anthropicVersion = "vertex-2023-10-16"

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, request *http.Request) (string, http.Header, io.Reader, error) {
	if request == nil {
		return "", nil, nil, errors.New("request is nil")
	}

	claudeReq, err := anthropic.ConvertRequest(meta, request)
	if err != nil {
		return "", nil, nil, err
	}
	req := Request{
		AnthropicVersion: anthropicVersion,
		Messages:         claudeReq.Messages,
		System:           claudeReq.System,
		MaxTokens:        claudeReq.MaxTokens,
		Temperature:      claudeReq.Temperature,
		TopP:             claudeReq.TopP,
		TopK:             claudeReq.TopK,
		Stream:           claudeReq.Stream,
		Tools:            claudeReq.Tools,
		ToolChoice:       claudeReq.ToolChoice,
	}
	data, err := sonic.Marshal(req)
	if err != nil {
		return "", nil, nil, err
	}
	return http.MethodPost, nil, bytes.NewReader(data), nil
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *relaymodel.Usage, err *relaymodel.ErrorWithStatusCode) {
	if utils.IsStreamResponse(resp) {
		usage, err = anthropic.StreamHandler(meta, c, resp)
	} else {
		usage, err = anthropic.Handler(meta, c, resp)
	}
	return
}
