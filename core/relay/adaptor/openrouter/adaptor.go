package openrouter

import (
	"errors"
	"net/http"

	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://openrouter.ai/api/v1"

func (a *Adaptor) DefaultBaseURL() string {
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
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			usage, err = openai.StreamHandler(meta, c, resp, streamPreHandler)
		} else {
			usage, err = openai.Handler(meta, c, resp, handlerPreHandler)
		}
	default:
		result, err := openai.DoResponse(meta, store, c, resp)
		if err != nil {
			return nil, err
		}

		usage = result.Usage()
	}

	if err != nil {
		return nil, err
	}

	return adaptor.NewSyncUsage(usage), nil
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme: "The `reasoning` field is converted to `reasoning_content`\nGemini support",
		Models: openai.ModelList,
	}
}
