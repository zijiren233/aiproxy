package vertexai

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/anthropic"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
	"github.com/pkg/errors"
)

var ModelList = []*model.ModelConfig{
	{
		Model: "claude-3-haiku@20240307",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-sonnet@20240229",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-opus@20240229",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-5-sonnet@20240620",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-5-sonnet-v2@20241022",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
	{
		Model: "claude-3-5-haiku@20241022",
		Type:  mode.ChatCompletions,
		Owner: model.ModelOwnerAnthropic,
	},
}

const anthropicVersion = "vertex-2023-10-16"

type Adaptor struct{}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, request *http.Request) (string, http.Header, io.Reader, error) {
	if request == nil {
		return "", nil, nil, errors.New("request is nil")
	}
	switch meta.Mode {
	case mode.ChatCompletions:
		claudeReq, err := anthropic.OpenAIConvertRequest(meta, request)
		if err != nil {
			return "", nil, nil, err
		}
		req := Request{
			AnthropicVersion: anthropicVersion,
			Request:          claudeReq,
		}
		req.Model = ""
		data, err := sonic.Marshal(req)
		if err != nil {
			return "", nil, nil, err
		}
		return http.MethodPost, nil, bytes.NewReader(data), nil
	case mode.Anthropic:
		reqBody, err := common.GetRequestBody(request)
		if err != nil {
			return "", nil, nil, err
		}
		node, err := sonic.Get(reqBody)
		if err != nil {
			return "", nil, nil, err
		}
		_, err = node.Unset("model")
		if err != nil {
			return "", nil, nil, err
		}
		_, err = node.Set("anthropic_version", ast.NewString(anthropicVersion))
		if err != nil {
			return "", nil, nil, err
		}
		data, err := node.MarshalJSON()
		if err != nil {
			return "", nil, nil, err
		}
		return http.MethodPost, nil, bytes.NewReader(data), nil
	default:
		return "", nil, nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			usage, err = anthropic.OpenAIStreamHandler(meta, c, resp)
		} else {
			usage, err = anthropic.OpenAIHandler(meta, c, resp)
		}
	case mode.Anthropic:
		if utils.IsStreamResponse(resp) {
			usage, err = anthropic.StreamHandler(meta, c, resp)
		} else {
			usage, err = anthropic.Handler(meta, c, resp)
		}
	default:
		return nil, openai.ErrorWrapperWithMessage(fmt.Sprintf("unsupported mode: %s", meta.Mode), "unsupported_mode", http.StatusBadRequest)
	}
	return
}
