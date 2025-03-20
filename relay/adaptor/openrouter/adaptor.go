package openrouter

import (
	"errors"
	"net/http"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/mode"
	relaymodel "github.com/labring/aiproxy/relay/model"
	"github.com/labring/aiproxy/relay/utils"
)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://openrouter.ai/api/v1"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

// choices.[*].delta.reasoning -> choices.[*].delta.reasoning_content
func streamPreHandler(_ *meta.Meta, node *ast.Node) error {
	choicesNode := node.Get("choices")
	nodes, err := choicesNode.ArrayUseNode()
	if err != nil {
		return err
	}
	for index, choice := range nodes {
		deltaNode := choice.Get("delta")
		reasoningString, err := deltaNode.Get("reasoning").String()
		if err != nil {
			if errors.Is(err, ast.ErrNotExist) {
				continue
			}
			return err
		}
		_, err = deltaNode.Set("reasoning_content", ast.NewString(reasoningString))
		if err != nil {
			return err
		}
		_, err = deltaNode.Unset("reasoning")
		if err != nil {
			return err
		}
		_, err = choicesNode.SetByIndex(index, choice)
		if err != nil {
			return err
		}
	}
	return nil
}

// choices.[*].message.reasoning -> choices.[*].message.reasoning_content
func handlerPreHandler(_ *meta.Meta, node *ast.Node) error {
	choicesNode := node.Get("choices")
	nodes, err := choicesNode.ArrayUseNode()
	if err != nil {
		return err
	}
	for index, choice := range nodes {
		messageNode := choice.Get("message")
		reasoningString, err := messageNode.Get("reasoning").String()
		if err != nil {
			if errors.Is(err, ast.ErrNotExist) {
				continue
			}
			return err
		}
		_, err = messageNode.Set("reasoning_content", ast.NewString(reasoningString))
		if err != nil {
			return err
		}
		_, err = messageNode.Unset("reasoning")
		if err != nil {
			return err
		}
		_, err = choicesNode.SetByIndex(index, choice)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *relaymodel.Usage, err *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			usage, err = openai.StreamHandler(meta, c, resp, streamPreHandler)
		} else {
			usage, err = openai.Handler(meta, c, resp, handlerPreHandler)
		}
	default:
		return openai.DoResponse(meta, c, resp)
	}
	return usage, err
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return openai.ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "openrouter"
}
