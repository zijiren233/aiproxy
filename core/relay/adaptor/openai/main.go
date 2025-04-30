package openai

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/common/render"
	"github.com/labring/aiproxy/core/common/splitter"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

const (
	DataPrefix       = "data:"
	Done             = "[DONE]"
	DataPrefixLength = len(DataPrefix)
)

var (
	DataPrefixBytes = conv.StringToBytes(DataPrefix)
	DoneBytes       = conv.StringToBytes(Done)
)

const scannerBufferSize = 256 * 1024

var scannerBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, scannerBufferSize)
		return &buf
	},
}

//nolint:forcetypeassert
func GetScannerBuffer() *[]byte {
	return scannerBufferPool.Get().(*[]byte)
}

func PutScannerBuffer(buf *[]byte) {
	if cap(*buf) != scannerBufferSize {
		return
	}
	scannerBufferPool.Put(buf)
}

func GetUsageOrChatChoicesResponseFromNode(node *ast.Node) (*relaymodel.Usage, []*relaymodel.ChatCompletionsStreamResponseChoice, error) {
	var usage *relaymodel.Usage
	usageNode, err := node.Get("usage").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(usageNode, &usage)
		if err != nil {
			return nil, nil, err
		}
	}

	if usage != nil {
		return usage, nil, nil
	}

	var choices []*relaymodel.ChatCompletionsStreamResponseChoice
	choicesNode, err := node.Get("choices").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(choicesNode, &choices)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, choices, nil
}

type PreHandler func(meta *meta.Meta, node *ast.Node) error

func StreamHandler(meta *meta.Meta, c *gin.Context, resp *http.Response, preHandler PreHandler) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseText := strings.Builder{}

	scanner := bufio.NewScanner(resp.Body)
	buf := GetScannerBuffer()
	defer PutScannerBuffer(buf)
	scanner.Buffer(*buf, cap(*buf))

	var usage *relaymodel.Usage

	hasReasoningContent := false
	var thinkSplitter *splitter.Splitter
	if meta.ChannelConfig.SplitThink {
		thinkSplitter = splitter.NewThinkSplitter()
	}

	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < DataPrefixLength { // ignore blank line or wrong format
			continue
		}
		if !slices.Equal(data[:DataPrefixLength], DataPrefixBytes) {
			continue
		}
		data = bytes.TrimSpace(data[DataPrefixLength:])
		if slices.Equal(data, DoneBytes) {
			break
		}

		node, err := sonic.Get(data)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}
		if preHandler != nil {
			err := preHandler(meta, &node)
			if err != nil {
				log.Error("error pre handler: " + err.Error())
				continue
			}
		}
		u, ch, err := GetUsageOrChatChoicesResponseFromNode(&node)
		if err != nil {
			log.Error("error unmarshalling stream response: " + err.Error())
			continue
		}
		if u != nil {
			usage = u
			responseText.Reset()
		}
		for _, choice := range ch {
			if usage == nil {
				if choice.Text != "" {
					responseText.WriteString(choice.Text)
				} else {
					responseText.WriteString(choice.Delta.StringContent())
				}
			}
			if choice.Delta.ReasoningContent != "" {
				hasReasoningContent = true
			}
		}

		_, err = node.Set("model", ast.NewString(meta.OriginModel))
		if err != nil {
			log.Error("error set model: " + err.Error())
		}

		if meta.ChannelConfig.SplitThink && !hasReasoningContent {
			respMap, err := node.Map()
			if err != nil {
				log.Error("error get node map: " + err.Error())
				continue
			}
			StreamSplitThink(respMap, thinkSplitter, func(data map[string]any) {
				_ = render.ObjectData(c, data)
			})
			continue
		}

		_ = render.ObjectData(c, &node)
	}

	if err := scanner.Err(); err != nil {
		log.Error("error reading stream: " + err.Error())
	}

	if usage == nil || (usage.TotalTokens == 0 && responseText.Len() > 0) {
		usage = ResponseText2Usage(
			responseText.String(),
			meta.ActualModel,
			int64(meta.RequestUsage.InputTokens),
		)
		_ = render.ObjectData(c, &relaymodel.ChatCompletionsStreamResponse{
			ID:      ChatCompletionID(),
			Model:   meta.OriginModel,
			Object:  relaymodel.ChatCompletionChunk,
			Created: time.Now().Unix(),
			Choices: []*relaymodel.ChatCompletionsStreamResponseChoice{},
			Usage:   usage,
		})
	} else if usage.TotalTokens != 0 && usage.PromptTokens == 0 { // some channels don't return prompt tokens & completion tokens
		usage.PromptTokens = int64(meta.RequestUsage.InputTokens)
		usage.CompletionTokens = usage.TotalTokens - int64(meta.RequestUsage.InputTokens)
	}

	render.Done(c)

	return usage.ToModelUsage(), nil
}

// renderCallback maybe reuse data, so don't modify data
func StreamSplitThink(data map[string]any, thinkSplitter *splitter.Splitter, renderCallback func(data map[string]any)) {
	choices, ok := data["choices"].([]any)
	// only support one choice
	if !ok || len(choices) != 1 {
		renderCallback(data)
		return
	}
	choice := choices[0]
	choiceMap, ok := choice.(map[string]any)
	if !ok {
		renderCallback(data)
		return
	}
	delta, ok := choiceMap["delta"].(map[string]any)
	if !ok {
		renderCallback(data)
		return
	}
	content, ok := delta["content"].(string)
	if !ok {
		renderCallback(data)
		return
	}
	think, remaining := thinkSplitter.Process(conv.StringToBytes(content))
	if len(think) == 0 && len(remaining) == 0 {
		delta["content"] = ""
		delete(delta, "reasoning_content")
		renderCallback(data)
		return
	}
	if len(think) > 0 {
		delta["content"] = ""
		delta["reasoning_content"] = conv.BytesToString(think)
		renderCallback(data)
	}
	if len(remaining) > 0 {
		delta["content"] = conv.BytesToString(remaining)
		delete(delta, "reasoning_content")
		renderCallback(data)
	}
}

func StreamSplitThinkModeld(data *relaymodel.ChatCompletionsStreamResponse, thinkSplitter *splitter.Splitter, renderCallback func(data *relaymodel.ChatCompletionsStreamResponse)) {
	choices := data.Choices
	// only support one choice
	if len(data.Choices) != 1 {
		renderCallback(data)
		return
	}
	choice := choices[0]
	content, ok := choice.Delta.Content.(string)
	if !ok {
		renderCallback(data)
		return
	}
	think, remaining := thinkSplitter.Process(conv.StringToBytes(content))
	if len(think) == 0 && len(remaining) == 0 {
		choice.Delta.Content = ""
		choice.Delta.ReasoningContent = ""
		renderCallback(data)
		return
	}
	if len(think) > 0 {
		choice.Delta.Content = ""
		choice.Delta.ReasoningContent = conv.BytesToString(think)
		renderCallback(data)
	}
	if len(remaining) > 0 {
		choice.Delta.Content = conv.BytesToString(remaining)
		choice.Delta.ReasoningContent = ""
		renderCallback(data)
	}
}

func SplitThink(data map[string]any) {
	choices, ok := data["choices"].([]any)
	if !ok {
		return
	}
	for _, choice := range choices {
		choiceMap, ok := choice.(map[string]any)
		if !ok {
			continue
		}
		message, ok := choiceMap["message"].(map[string]any)
		if !ok {
			continue
		}
		content, ok := message["content"].(string)
		if !ok {
			continue
		}
		think, remaining := splitter.NewThinkSplitter().Process(conv.StringToBytes(content))
		message["reasoning_content"] = conv.BytesToString(think)
		message["content"] = conv.BytesToString(remaining)
	}
}

func SplitThinkModeld(data *relaymodel.TextResponse) {
	choices := data.Choices
	for _, choice := range choices {
		content, ok := choice.Message.Content.(string)
		if !ok {
			continue
		}
		think, remaining := splitter.NewThinkSplitter().Process(conv.StringToBytes(content))
		choice.Message.ReasoningContent = conv.BytesToString(think)
		choice.Message.Content = conv.BytesToString(remaining)
	}
}

func GetUsageOrChoicesResponseFromNode(node *ast.Node) (*relaymodel.Usage, []*relaymodel.TextResponseChoice, error) {
	var usage *relaymodel.Usage
	usageNode, err := node.Get("usage").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(usageNode, &usage)
		if err != nil {
			return nil, nil, err
		}
	}

	if usage != nil {
		return usage, nil, nil
	}

	var choices []*relaymodel.TextResponseChoice
	choicesNode, err := node.Get("choices").Raw()
	if err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return nil, nil, err
		}
	} else {
		err = sonic.UnmarshalString(choicesNode, &choices)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, choices, nil
}

func Handler(meta *meta.Meta, c *gin.Context, resp *http.Response, preHandler PreHandler) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}

	node, err := sonic.Get(responseBody)
	if err != nil {
		return nil, ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if preHandler != nil {
		err := preHandler(meta, &node)
		if err != nil {
			return nil, ErrorWrapper(err, "pre_handler_failed", http.StatusInternalServerError)
		}
	}
	usage, choices, err := GetUsageOrChoicesResponseFromNode(&node)
	if err != nil {
		return nil, ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}

	if usage == nil || usage.TotalTokens == 0 || (usage.PromptTokens == 0 && usage.CompletionTokens == 0) {
		var completionTokens int64
		for _, choice := range choices {
			if choice.Text != "" {
				completionTokens += CountTokenText(choice.Text, meta.ActualModel)
				continue
			}
			completionTokens += CountTokenText(choice.Message.StringContent(), meta.ActualModel)
		}
		usage = &relaymodel.Usage{
			PromptTokens:     int64(meta.RequestUsage.InputTokens),
			CompletionTokens: completionTokens,
			TotalTokens:      int64(meta.RequestUsage.InputTokens) + completionTokens,
		}
		_, err = node.Set("usage", ast.NewAny(usage))
		if err != nil {
			return usage.ToModelUsage(), ErrorWrapper(err, "set_usage_failed", http.StatusInternalServerError)
		}
	} else if usage.TotalTokens != 0 && usage.PromptTokens == 0 { // some channels don't return prompt tokens & completion tokens
		usage.PromptTokens = int64(meta.RequestUsage.InputTokens)
		usage.CompletionTokens = usage.TotalTokens - int64(meta.RequestUsage.InputTokens)
		_, err = node.Set("usage", ast.NewAny(usage))
		if err != nil {
			return usage.ToModelUsage(), ErrorWrapper(err, "set_usage_failed", http.StatusInternalServerError)
		}
	}

	_, err = node.Set("model", ast.NewString(meta.OriginModel))
	if err != nil {
		return usage.ToModelUsage(), ErrorWrapper(err, "set_model_failed", http.StatusInternalServerError)
	}

	if meta.ChannelConfig.SplitThink {
		respMap, err := node.Map()
		if err != nil {
			return usage.ToModelUsage(), ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
		}
		SplitThink(respMap)
		c.JSON(http.StatusOK, respMap)
		return usage.ToModelUsage(), nil
	}

	newData, err := sonic.Marshal(&node)
	if err != nil {
		return usage.ToModelUsage(), ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}

	_, err = c.Writer.Write(newData)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	return usage.ToModelUsage(), nil
}
