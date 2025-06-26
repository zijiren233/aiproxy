package streamfake

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/plugin"
	"github.com/labring/aiproxy/core/relay/plugin/noop"
)

var _ plugin.Plugin = (*StreamFake)(nil)

// StreamFake implements the stream fake functionality
type StreamFake struct {
	noop.Noop
}

// NewStreamFakePlugin creates a new stream fake plugin instance
func NewStreamFakePlugin() plugin.Plugin {
	return &StreamFake{}
}

// Constants for metadata keys
const (
	fakeStreamKey = "fake_stream"
)

// getConfig retrieves the plugin configuration
func (p *StreamFake) getConfig(meta *meta.Meta) (*Config, error) {
	pluginConfig := &Config{}
	if err := meta.ModelConfig.LoadPluginConfig("stream-fake", pluginConfig); err != nil {
		return nil, err
	}
	return pluginConfig, nil
}

// ConvertRequest modifies the request to enable streaming if it's originally non-streaming
func (p *StreamFake) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	req *http.Request,
	do adaptor.ConvertRequest,
) (adaptor.ConvertResult, error) {
	// Only process chat completions
	if meta.Mode != mode.ChatCompletions {
		return do.ConvertRequest(meta, store, req)
	}

	// Check if stream fake is enabled
	// pluginConfig, err := p.getConfig(meta)
	// if err != nil || !pluginConfig.Enable {
	// 	return do.ConvertRequest(meta, store, req)
	// }

	body, err := common.GetRequestBodyReusable(req)
	if err != nil {
		return adaptor.ConvertResult{}, fmt.Errorf("failed to read request body: %w", err)
	}

	node, err := sonic.Get(body)
	if err != nil {
		return do.ConvertRequest(meta, store, req)
	}

	stream, _ := node.Get("stream").Bool()
	if stream {
		// Already streaming, no need to fake
		return do.ConvertRequest(meta, store, req)
	}

	meta.Set(fakeStreamKey, true)

	// Modify request to enable streaming
	node.Set("stream", ast.NewBool(true))

	// Create new request body
	modifiedBody, err := node.MarshalJSON()
	if err != nil {
		return do.ConvertRequest(meta, store, req)
	}

	// Update the request
	common.SetRequestBody(req, modifiedBody)
	defer common.SetRequestBody(req, body)

	return do.ConvertRequest(meta, store, req)
}

// DoResponse handles the response processing to collect streaming data and convert back to non-streaming
func (p *StreamFake) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
	do adaptor.DoResponse,
) (model.Usage, adaptor.Error) {
	// Only process chat completions
	if meta.Mode != mode.ChatCompletions {
		return do.DoResponse(meta, store, c, resp)
	}

	// Check if this is a fake stream request
	isFakeStream, ok := meta.Get(fakeStreamKey)
	if !ok {
		return do.DoResponse(meta, store, c, resp)
	}
	isFakeStreamBool, ok := isFakeStream.(bool)
	if !ok || !isFakeStreamBool {
		return do.DoResponse(meta, store, c, resp)
	}

	return p.handleFakeStreamResponse(meta, store, c, resp, do)
}

// handleFakeStreamResponse processes the streaming response and converts it back to non-streaming
func (p *StreamFake) handleFakeStreamResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
	do adaptor.DoResponse,
) (model.Usage, adaptor.Error) {
	// Create a custom response writer to collect streaming data
	rw := &fakeStreamResponseWriter{
		ResponseWriter: c.Writer,
	}
	c.Writer = rw
	defer func() {
		c.Writer = rw.ResponseWriter
	}()

	// Process the streaming response
	usage, relayErr := do.DoResponse(meta, store, c, resp)
	if relayErr != nil {
		return usage, relayErr
	}

	// Convert collected streaming chunks to non-streaming response
	respBody, err := rw.convertToNonStream()
	if err != nil {
		return usage, relayErr
	}

	// Set appropriate headers for non-streaming response
	c.Header("Content-Type", "application/json")
	c.Header("Content-Length", strconv.Itoa(len(respBody)))

	// Remove streaming-specific headers
	c.Header("Cache-Control", "")
	c.Header("Connection", "")
	c.Header("Transfer-Encoding", "")
	c.Header("X-Accel-Buffering", "")

	// Write the non-streaming response
	_, _ = rw.ResponseWriter.Write(respBody)

	return usage, nil
}

// fakeStreamResponseWriter captures streaming response data
type fakeStreamResponseWriter struct {
	gin.ResponseWriter

	lastChunk        *ast.Node
	usageNode        *ast.Node
	contentBuilder   bytes.Buffer
	reasoningContent bytes.Buffer
	finishReason     relaymodel.FinishReason
	logprobsContent  []ast.Node
	toolCalls        []*relaymodel.ToolCall
}

// ignore flush
func (rw *fakeStreamResponseWriter) Flush() {}

func (rw *fakeStreamResponseWriter) Write(b []byte) (int, error) {
	// Parse streaming data
	_ = rw.parseStreamingData(b)

	return len(b), nil
}

func (rw *fakeStreamResponseWriter) WriteString(s string) (int, error) {
	return rw.Write(conv.StringToBytes(s))
}

// parseStreamingData extracts individual chunks from streaming response
func (rw *fakeStreamResponseWriter) parseStreamingData(data []byte) error {
	node, err := sonic.Get(data)
	if err != nil {
		return err
	}
	rw.lastChunk = &node
	usageNode := node.Get("usage")
	if err := usageNode.Check(); err != nil {
		if !errors.Is(err, ast.ErrNotExist) {
			return err
		}
	} else {
		rw.usageNode = usageNode
	}

	choicesNode := node.Get("choices")
	if err := choicesNode.Check(); err != nil {
		return err
	}

	choicesNode.ForEach(func(_ ast.Sequence, choiceNode *ast.Node) bool {
		deltaNode := choiceNode.Get("delta")
		if err := deltaNode.Check(); err != nil {
			return true
		}
		content, err := deltaNode.Get("content").String()
		if err == nil {
			rw.contentBuilder.WriteString(content)
		}
		reasoningContent, err := deltaNode.Get("reasoning_content").String()
		if err == nil {
			rw.reasoningContent.WriteString(reasoningContent)
		}
		deltaNode.Get("tool_calls").ForEach(func(_ ast.Sequence, toolCallNode *ast.Node) bool {
			toolCallRaw, err := toolCallNode.Raw()
			if err != nil {
				return true
			}
			var toolCall relaymodel.ToolCall
			if err := sonic.UnmarshalString(toolCallRaw, &toolCall); err != nil {
				return true
			}
			rw.toolCalls = mergeToolCalls(rw.toolCalls, &toolCall)
			return true
		})
		finishReason, err := choiceNode.Get("finish_reason").String()
		if err == nil && finishReason != "" {
			rw.finishReason = finishReason
		}
		logprobsContentNode := choiceNode.Get("logprobs").Get("content")
		if err := logprobsContentNode.Check(); err == nil {
			l, err := logprobsContentNode.Len()
			if err != nil {
				return true
			}
			rw.logprobsContent = slices.Grow(rw.logprobsContent, l)
			logprobsContentNode.ForEach(func(_ ast.Sequence, logprobsContentNode *ast.Node) bool {
				rw.logprobsContent = append(rw.logprobsContent, *logprobsContentNode)
				return true
			})
		}
		return true
	})

	return nil
}

func (rw *fakeStreamResponseWriter) convertToNonStream() ([]byte, error) {
	lastChunk := rw.lastChunk
	if lastChunk == nil {
		return nil, errors.New("last chunk is nil")
	}

	lastChunk.Set("object", ast.NewString(relaymodel.ChatCompletionObject))
	if rw.usageNode != nil {
		lastChunk.Set("usage", *rw.usageNode)
	}

	message := map[string]any{
		"role":    "assistant",
		"content": rw.contentBuilder.String(),
	}

	reasoningContent := rw.reasoningContent.String()
	if reasoningContent != "" {
		message["reasoning_content"] = reasoningContent
	}

	if len(rw.toolCalls) > 0 {
		slices.SortFunc(rw.toolCalls, func(a, b *relaymodel.ToolCall) int {
			return a.Index - b.Index
		})
		message["tool_calls"] = rw.toolCalls
	}
	if len(rw.logprobsContent) > 0 {
		message["logprobs"] = map[string]any{
			"content": rw.logprobsContent,
		}
	}

	lastChunk.SetAny("choices", []any{
		map[string]any{
			"index":         0,
			"message":       message,
			"finish_reason": rw.finishReason,
		},
	})

	return lastChunk.MarshalJSON()
}

func mergeToolCalls(
	oldToolCalls []*relaymodel.ToolCall,
	newToolCall *relaymodel.ToolCall,
) []*relaymodel.ToolCall {
	findedToolCallIndex := slices.IndexFunc(oldToolCalls, func(t *relaymodel.ToolCall) bool {
		return t.Index == newToolCall.Index
	})
	if findedToolCallIndex != -1 {
		oldToolCall := oldToolCalls[findedToolCallIndex]
		oldToolCalls[findedToolCallIndex] = mergeToolCall(oldToolCall, newToolCall)
	} else {
		oldToolCalls = append(oldToolCalls, newToolCall)
	}
	return oldToolCalls
}

func mergeToolCall(oldToolCall, newToolCall *relaymodel.ToolCall) *relaymodel.ToolCall {
	if oldToolCall == nil {
		return newToolCall
	}

	if newToolCall == nil {
		return oldToolCall
	}

	merged := &relaymodel.ToolCall{
		Index:    oldToolCall.Index,
		ID:       oldToolCall.ID,
		Type:     oldToolCall.Type,
		Function: oldToolCall.Function,
	}

	merged.Function.Arguments += newToolCall.Function.Arguments

	return merged
}
