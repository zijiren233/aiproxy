//nolint:testpackage
package ali

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func TestAdaptorSupportModeResponses(t *testing.T) {
	adaptor := &Adaptor{}

	supportedModes := []mode.Mode{
		mode.Responses,
		mode.ResponsesGet,
		mode.ResponsesDelete,
		mode.ResponsesCancel,
		mode.ResponsesInputItems,
	}
	for _, m := range supportedModes {
		if !adaptor.SupportMode(&meta.Meta{Mode: m}) {
			t.Fatalf("expected mode %s to be supported", m)
		}
	}
}

func TestAdaptorGetRequestURLResponses(t *testing.T) {
	adaptor := &Adaptor{}
	channel := &coremodel.Channel{BaseURL: "https://dashscope.aliyuncs.com"}

	tests := []struct {
		name       string
		mode       mode.Mode
		responseID string
		wantMethod string
		wantURL    string
	}{
		{
			name:       "responses create",
			mode:       mode.Responses,
			wantMethod: http.MethodPost,
			wantURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1/responses",
		},
		{
			name:       "responses get",
			mode:       mode.ResponsesGet,
			responseID: "resp_123",
			wantMethod: http.MethodGet,
			wantURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1/responses/resp_123",
		},
		{
			name:       "responses delete",
			mode:       mode.ResponsesDelete,
			responseID: "resp_123",
			wantMethod: http.MethodDelete,
			wantURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1/responses/resp_123",
		},
		{
			name:       "responses cancel",
			mode:       mode.ResponsesCancel,
			responseID: "resp_123",
			wantMethod: http.MethodPost,
			wantURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1/responses/resp_123/cancel",
		},
		{
			name:       "responses input items",
			mode:       mode.ResponsesInputItems,
			responseID: "resp_123",
			wantMethod: http.MethodGet,
			wantURL:    "https://dashscope.aliyuncs.com/compatible-mode/v1/responses/resp_123/input_items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := meta.NewMeta(
				channel,
				tt.mode,
				"qwen-plus",
				coremodel.ModelConfig{},
				meta.WithResponseID(tt.responseID),
			)

			got, err := adaptor.GetRequestURL(m, nil, nil)
			if err != nil {
				t.Fatalf("GetRequestURL returned error: %v", err)
			}

			if got.Method != tt.wantMethod {
				t.Fatalf("expected method %s, got %s", tt.wantMethod, got.Method)
			}

			if got.URL != tt.wantURL {
				t.Fatalf("expected URL %s, got %s", tt.wantURL, got.URL)
			}
		})
	}
}

func TestAdaptorGetRequestURLMultimodalEmbeddings(t *testing.T) {
	adaptor := &Adaptor{}
	channel := &coremodel.Channel{BaseURL: "https://dashscope.aliyuncs.com"}

	tests := []struct {
		name    string
		origin  string
		actual  string
		wantURL string
	}{
		{
			name:    "vl model uses multimodal embedding endpoint",
			origin:  "qwen3-vl-embedding",
			actual:  "mapped-model",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding",
		},
		{
			name:    "multimodal model uses multimodal embedding endpoint",
			origin:  "multimodal-embedding-v1",
			actual:  "mapped-model",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding",
		},
		{
			name:    "vision model uses multimodal embedding endpoint",
			origin:  "tongyi-embedding-vision-plus",
			actual:  "mapped-model",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding",
		},
		{
			name:    "actual multimodal model uses multimodal embedding endpoint",
			origin:  "alias-model",
			actual:  "qwen3-vl-embedding",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding",
		},
		{
			name:    "text embedding model keeps compatible endpoint",
			origin:  "text-embedding-v4",
			actual:  "text-embedding-v4",
			wantURL: "https://dashscope.aliyuncs.com/compatible-mode/v1/embeddings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := meta.NewMeta(channel, mode.Embeddings, tt.origin, coremodel.ModelConfig{})
			m.ActualModel = tt.actual

			got, err := adaptor.GetRequestURL(m, nil, nil)
			if err != nil {
				t.Fatalf("GetRequestURL returned error: %v", err)
			}

			if got.Method != http.MethodPost {
				t.Fatalf("expected method %s, got %s", http.MethodPost, got.Method)
			}

			if got.URL != tt.wantURL {
				t.Fatalf("expected URL %s, got %s", tt.wantURL, got.URL)
			}
		})
	}
}

func TestAdaptorConvertRequestResponses(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.Responses,
		"qwen-plus",
		coremodel.ModelConfig{},
	)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/responses",
		strings.NewReader(`{"model":"qwen-plus","input":"hello","stream":true}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var responseReq relaymodel.CreateResponseRequest
	if err := json.Unmarshal(body, &responseReq); err != nil {
		t.Fatalf("failed to unmarshal converted body: %v", err)
	}

	if responseReq.Model != "qwen-plus" {
		t.Fatalf("expected model qwen-plus, got %s", responseReq.Model)
	}

	if !responseReq.Stream {
		t.Fatal("expected stream to remain enabled")
	}
}

func TestAdaptorConvertRequestMultimodalEmbeddings(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.Embeddings,
		"qwen3-vl-embedding",
		coremodel.ModelConfig{},
	)
	m.ActualModel = "mapped-vl-embedding"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/embeddings",
		strings.NewReader(`{
			"model": "qwen3-vl-embedding",
			"input": [
				{
					"type": "image_url",
					"image_url": {
						"url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg"
					}
				},
				{
					"type": "text",
					"text": "请描述这张图片"
				},
				"plain text",
				{
					"image": "https://example.com/image.jpg"
				}
			],
			"dimensions": 1024,
			"encoding_format": "float"
		}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body %s: %v", string(body), err)
	}

	if payload["model"] != "mapped-vl-embedding" {
		t.Fatalf("expected mapped model, got %#v", payload["model"])
	}

	if _, exists := payload["encoding_format"]; exists {
		t.Fatal("expected encoding_format to be removed")
	}

	if _, exists := payload["dimensions"]; exists {
		t.Fatal("expected dimensions to be removed")
	}

	parameters, ok := payload["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("expected parameters object, got %#v", payload["parameters"])
	}

	if dimension, ok := parameters["dimension"].(float64); !ok || int(dimension) != 1024 {
		t.Fatalf("expected parameters.dimension=1024, got %#v", parameters["dimension"])
	}

	input, ok := payload["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input object, got %#v", payload["input"])
	}

	contents, ok := input["contents"].([]any)
	if !ok {
		t.Fatalf("expected input.contents array, got %#v", input["contents"])
	}

	if len(contents) != 4 {
		t.Fatalf("expected 4 contents, got %d", len(contents))
	}

	assertContentString(t, contents[0], "image", "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg")
	assertContentString(t, contents[1], "text", "请描述这张图片")
	assertContentString(t, contents[2], "text", "plain text")
	assertContentString(t, contents[3], "image", "https://example.com/image.jpg")
}

func TestAdaptorDoResponseResponsesDeleteNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &Adaptor{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}

	_, err := adaptor.DoResponse(
		&meta.Meta{Mode: mode.ResponsesDelete},
		nil,
		ctx,
		resp,
	)
	if err != nil {
		t.Fatalf("DoResponse returned error: %v", err)
	}

	if ctx.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, ctx.Writer.Status())
	}
}

func TestAdaptorDoResponseMultimodalEmbeddings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &Adaptor{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/embeddings",
		nil,
	)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"output": {
				"embeddings": [
					{
						"index": 0,
						"embedding": [0.1, 0.2],
						"type": "text"
					},
					{
						"index": 1,
						"embedding": [0.3, 0.4],
						"type": "image"
					}
				]
			},
			"usage": {
				"input_tokens": 10,
				"input_tokens_details": {
					"image_tokens": 8,
					"text_tokens": 2
				},
				"output_tokens": 2,
				"total_tokens": 12
			},
			"request_id": "req_123"
		}`)),
	}

	result, err := adaptor.DoResponse(
		&meta.Meta{
			Mode:        mode.Embeddings,
			OriginModel: "qwen3-vl-embedding",
			ActualModel: "qwen3-vl-embedding",
		},
		nil,
		ctx,
		resp,
	)
	if err != nil {
		t.Fatalf("DoResponse returned error: %v", err)
	}

	if result.Usage.InputTokens != coremodel.ZeroNullInt64(10) {
		t.Fatalf("expected input tokens 10, got %d", result.Usage.InputTokens)
	}

	if result.Usage.ImageInputTokens != coremodel.ZeroNullInt64(8) {
		t.Fatalf("expected image input tokens 8, got %d", result.Usage.ImageInputTokens)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response body %s: %v", recorder.Body.String(), err)
	}

	if payload["object"] != "list" {
		t.Fatalf("expected object=list, got %#v", payload["object"])
	}

	if payload["model"] != "qwen3-vl-embedding" {
		t.Fatalf("expected model qwen3-vl-embedding, got %#v", payload["model"])
	}

	data, ok := payload["data"].([]any)
	if !ok || len(data) != 2 {
		t.Fatalf("expected 2 data items, got %#v", payload["data"])
	}

	first, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first data object, got %#v", data[0])
	}

	if first["object"] != "embedding" {
		t.Fatalf("expected first object=embedding, got %#v", first["object"])
	}

	usage, ok := payload["usage"].(map[string]any)
	if !ok {
		t.Fatalf("expected usage object, got %#v", payload["usage"])
	}

	if promptTokens, ok := usage["prompt_tokens"].(float64); !ok || int(promptTokens) != 10 {
		t.Fatalf("expected usage.prompt_tokens=10, got %#v", usage["prompt_tokens"])
	}

	details, ok := usage["prompt_tokens_details"].(map[string]any)
	if !ok {
		t.Fatalf(
			"expected usage.prompt_tokens_details object, got %#v",
			usage["prompt_tokens_details"],
		)
	}

	if imageTokens, ok := details["image_tokens"].(float64); !ok || int(imageTokens) != 8 {
		t.Fatalf("expected image_tokens=8, got %#v", details["image_tokens"])
	}
}

func TestAdaptorConvertGeminiRequestReasoning(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.Gemini,
		"glm-4.5",
		coremodel.ModelConfig{},
	)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1beta/models/gemini-pro:generateContent",
		strings.NewReader(`{
			"generationConfig": {
				"thinkingConfig": {
					"thinkingBudget": 2048,
					"includeThoughts": true
				}
			},
			"contents": [{"role":"user","parts":[{"text":"hello"}]}]
		}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body: %v", err)
	}

	enableThinking, ok := payload["enable_thinking"].(bool)
	if !ok || !enableThinking {
		t.Fatalf("expected enable_thinking=true, got %#v", payload["enable_thinking"])
	}

	thinkingBudget, ok := payload["thinking_budget"].(float64)
	if !ok || int(thinkingBudget) != 2048 {
		t.Fatalf("expected thinking_budget=2048, got %#v", payload["thinking_budget"])
	}

	if _, exists := payload["reasoning_effort"]; exists {
		t.Fatal("expected reasoning_effort to be removed")
	}
}

func assertContentString(t *testing.T, got any, key, want string) {
	t.Helper()

	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected content object, got %T", got)
	}

	if len(gotMap) != 1 {
		t.Fatalf("expected content to have 1 key, got %#v", gotMap)
	}

	if gotMap[key] != want {
		t.Fatalf("expected %s=%q, got %#v", key, want, gotMap[key])
	}
}

func TestAdaptorConvertChatCompletionsReasoningNotClampedByMaxTokens(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.ChatCompletions,
		"glm-4.5",
		coremodel.ModelConfig{},
	)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{
			"model":"glm-4.5",
			"reasoning_effort":"high",
			"max_tokens":1000,
			"messages":[{"role":"user","content":"hello"}]
		}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body: %v", err)
	}

	enableThinking, ok := payload["enable_thinking"].(bool)
	if !ok || !enableThinking {
		t.Fatalf("expected enable_thinking=true, got %#v", payload["enable_thinking"])
	}

	thinkingBudget, ok := payload["thinking_budget"].(float64)
	if !ok || int(thinkingBudget) != 16384 {
		t.Fatalf("expected thinking_budget=16384, got %#v", payload["thinking_budget"])
	}
}

func TestAdaptorConvertChatCompletionsReasoningUsesOriginModelNameFirst(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.ChatCompletions,
		"glm-4.5",
		coremodel.ModelConfig{},
	)
	m.ActualModel = "mapped-upstream-model"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{
			"model":"glm-4.5",
			"reasoning_effort":"high",
			"messages":[{"role":"user","content":"hello"}]
		}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body: %v", err)
	}

	enableThinking, ok := payload["enable_thinking"].(bool)
	if !ok || !enableThinking {
		t.Fatalf("expected enable_thinking=true, got %#v", payload["enable_thinking"])
	}

	thinkingBudget, ok := payload["thinking_budget"].(float64)
	if !ok || int(thinkingBudget) != 16384 {
		t.Fatalf("expected thinking_budget=16384, got %#v", payload["thinking_budget"])
	}
}

func TestAdaptorConvertCompletionsReasoning(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.Completions,
		"glm-4.5",
		coremodel.ModelConfig{},
	)

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/completions",
		strings.NewReader(`{
			"model":"glm-4.5",
			"reasoning_effort":"low",
			"prompt":"hello"
		}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body: %v", err)
	}

	enableThinking, ok := payload["enable_thinking"].(bool)
	if !ok || !enableThinking {
		t.Fatalf("expected enable_thinking=true, got %#v", payload["enable_thinking"])
	}
}
