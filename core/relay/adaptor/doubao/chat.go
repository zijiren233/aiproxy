package doubao

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ConvertChatCompletionsRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	result, err := openai.ConvertChatCompletionsRequest(
		meta,
		req,
		false,
	)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if !strings.HasPrefix(meta.OriginModel, "deepseek-reasoner") {
		return result, nil
	}

	m := make(map[string]any)

	err = sonic.ConfigDefault.NewDecoder(result.Body).Decode(&m)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	messages, _ := m["messages"].([]any)
	if len(messages) == 0 {
		return adaptor.ConvertResult{}, errors.New("messages is empty")
	}

	sysMessage := relaymodel.Message{
		Role:    "system",
		Content: "回答前，都先用 <think></think> 输出你的思考过程。",
	}
	messages = append([]any{sysMessage}, messages...)
	m["messages"] = messages

	newBody, err := sonic.Marshal(m)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	header := result.Header
	header.Set("Content-Length", strconv.Itoa(len(newBody)))

	return adaptor.ConvertResult{
		Header: header,
		Body:   bytes.NewReader(newBody),
	}, nil
}

func newHandlerPreHandler(websearchCount *int64) func(_ *meta.Meta, node *ast.Node) error {
	return func(meta *meta.Meta, node *ast.Node) error {
		return handlerPreHandler(meta, node, websearchCount)
	}
}

// copy bot_usage.model_usage to usage
func handlerPreHandler(meta *meta.Meta, node *ast.Node, websearchCount *int64) error {
	if !strings.HasPrefix(meta.ActualModel, "bot-") {
		return nil
	}

	botUsageNode := node.Get("bot_usage")
	if botUsageNode.Check() != nil {
		return nil
	}

	modelUsageNode := botUsageNode.Get("model_usage").Index(0)
	if modelUsageNode.Check() != nil {
		return nil
	}

	_, err := node.SetAny("usage", modelUsageNode)
	if err != nil {
		return err
	}

	actionUsageNodes := botUsageNode.Get("action_usage")
	if actionUsageNodes.Check() != nil {
		return nil
	}

	return actionUsageNodes.ForEach(func(_ ast.Sequence, node *ast.Node) bool {
		if node.Check() != nil {
			return true
		}

		count, err := node.Get("count").Int64()
		if err != nil {
			return true
		}

		*websearchCount += count

		return true
	})
}
