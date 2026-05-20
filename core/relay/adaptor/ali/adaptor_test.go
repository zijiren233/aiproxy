//nolint:testpackage
package ali

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
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

func TestAdaptorGetRequestURLQwenImage(t *testing.T) {
	adaptor := &Adaptor{}
	channel := &coremodel.Channel{BaseURL: "https://dashscope.aliyuncs.com"}

	tests := []struct {
		name    string
		mode    mode.Mode
		model   string
		wantURL string
	}{
		{
			name:    "qwen image generations uses multimodal generation endpoint",
			mode:    mode.ImagesGenerations,
			model:   "qwen-image-2.0-pro",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
		},
		{
			name:    "qwen image edits uses multimodal generation endpoint",
			mode:    mode.ImagesEdits,
			model:   "qwen-image-edit-plus",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation",
		},
		{
			name:    "legacy image generation keeps async endpoint",
			mode:    mode.ImagesGenerations,
			model:   "stable-diffusion-xl",
			wantURL: "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := meta.NewMeta(channel, tt.mode, tt.model, coremodel.ModelConfig{})

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

func TestAdaptorConvertRequestQwenImageGeneration(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(nil, mode.ImagesGenerations, "qwen-image-plus", coremodel.ModelConfig{})
	m.ActualModel = "mapped-qwen-image"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{
			"model": "qwen-image-plus",
			"prompt": "draw a cat",
			"size": "1024x1024",
			"negative_prompt": "low quality",
			"prompt_extend": true,
			"watermark": false,
			"seed": 123,
			"quality": "high",
			"response_format": "b64_json"
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

	if payload["model"] != "mapped-qwen-image" {
		t.Fatalf("expected mapped model, got %#v", payload["model"])
	}

	input := payload["input"].(map[string]any)
	messages := input["messages"].([]any)
	message := messages[0].(map[string]any)
	contents := message["content"].([]any)
	assertContentString(t, contents[0], "text", "draw a cat")

	parameters := payload["parameters"].(map[string]any)
	if parameters["size"] != "1024*1024" {
		t.Fatalf("expected size 1024*1024, got %#v", parameters["size"])
	}

	if parameters["negative_prompt"] != "low quality" {
		t.Fatalf("expected negative_prompt, got %#v", parameters["negative_prompt"])
	}

	if parameters["prompt_extend"] != true {
		t.Fatalf("expected prompt_extend true, got %#v", parameters["prompt_extend"])
	}

	if parameters["watermark"] != false {
		t.Fatalf("expected watermark false, got %#v", parameters["watermark"])
	}

	if int64(parameters["seed"].(float64)) != 123 {
		t.Fatalf("expected seed 123, got %#v", parameters["seed"])
	}

	if _, ok := parameters["quality"]; ok {
		t.Fatalf("expected quality not to be forwarded, got %#v", parameters["quality"])
	}

	if _, ok := parameters["n"]; ok {
		t.Fatalf("expected n not to be forwarded for this model, got %#v", parameters["n"])
	}
}

func TestAdaptorConvertRequestQwenImageGenerationRejectsUnsupportedN(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(nil, mode.ImagesGenerations, "qwen-image-plus", coremodel.ModelConfig{})
	m.ActualModel = "qwen-image-plus"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{
			"model": "qwen-image-plus",
			"prompt": "draw a cat",
			"n": 2
		}`),
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	_, err = adaptor.ConvertRequest(m, nil, req)
	if err == nil || !strings.Contains(err.Error(), "n must be 1") {
		t.Fatalf("expected unsupported n error, got %v", err)
	}
}

func TestAdaptorConvertRequestQwenImageGenerationSupportsN(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(nil, mode.ImagesGenerations, "qwen-image-2.0-pro", coremodel.ModelConfig{})
	m.ActualModel = "qwen-image-2.0-pro"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{
			"model": "qwen-image-2.0-pro",
			"prompt": "draw a cat",
			"n": 2
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

	parameters := payload["parameters"].(map[string]any)
	if int(parameters["n"].(float64)) != 2 {
		t.Fatalf("expected n 2, got %#v", parameters["n"])
	}
}

func TestAdaptorConvertRequestQwenImageEdit(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(nil, mode.ImagesEdits, "qwen-image-edit-plus", coremodel.ModelConfig{})
	m.ActualModel = "mapped-qwen-edit"

	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)
	if err := writer.WriteField("prompt", "add a hat"); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}

	if err := writer.WriteField("size", "1024x1024"); err != nil {
		t.Fatalf("failed to write size: %v", err)
	}

	if err := writer.WriteField("negative_prompt", "blurry"); err != nil {
		t.Fatalf("failed to write negative_prompt: %v", err)
	}

	if err := writer.WriteField("prompt_extend", "true"); err != nil {
		t.Fatalf("failed to write prompt_extend: %v", err)
	}

	if err := writer.WriteField("watermark", "false"); err != nil {
		t.Fatalf("failed to write watermark: %v", err)
	}

	if err := writer.WriteField("seed", "456"); err != nil {
		t.Fatalf("failed to write seed: %v", err)
	}

	if err := writer.WriteField("response_format", "url"); err != nil {
		t.Fatalf("failed to write response_format: %v", err)
	}

	part, err := writer.CreateFormFile("image", "input.png")
	if err != nil {
		t.Fatalf("failed to create file part: %v", err)
	}

	_, _ = part.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	})

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/edits",
		body,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	convertedBody, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(convertedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body %s: %v", string(convertedBody), err)
	}

	if payload["model"] != "mapped-qwen-edit" {
		t.Fatalf("expected mapped model, got %#v", payload["model"])
	}

	input := payload["input"].(map[string]any)
	messages := input["messages"].([]any)
	message := messages[0].(map[string]any)
	contents := message["content"].([]any)
	assertContentHasPrefix(t, contents[0], "image", "data:image/png;base64,")
	assertContentString(t, contents[1], "text", "add a hat")

	parameters := payload["parameters"].(map[string]any)
	if parameters["size"] != "1024*1024" {
		t.Fatalf("expected size 1024*1024, got %#v", parameters["size"])
	}

	if parameters["negative_prompt"] != "blurry" {
		t.Fatalf("expected negative_prompt, got %#v", parameters["negative_prompt"])
	}

	if parameters["prompt_extend"] != true {
		t.Fatalf("expected prompt_extend true, got %#v", parameters["prompt_extend"])
	}

	if parameters["watermark"] != false {
		t.Fatalf("expected watermark false, got %#v", parameters["watermark"])
	}

	if int64(parameters["seed"].(float64)) != 456 {
		t.Fatalf("expected seed 456, got %#v", parameters["seed"])
	}
}

func TestAdaptorConvertRequestQwenImageEditAcceptsImageArrayField(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(nil, mode.ImagesEdits, "qwen-image-edit-plus", coremodel.ModelConfig{})
	m.ActualModel = "qwen-image-edit-plus"

	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)
	if err := writer.WriteField("prompt", "blend two images"); err != nil {
		t.Fatalf("failed to write prompt: %v", err)
	}

	for _, filename := range []string{"input1.png", "input2.png"} {
		part, err := writer.CreateFormFile("image[]", filename)
		if err != nil {
			t.Fatalf("failed to create file part: %v", err)
		}

		_, _ = part.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
			0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		})
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/edits",
		body,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	convertedBody, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(convertedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body %s: %v", string(convertedBody), err)
	}

	input := payload["input"].(map[string]any)
	messages := input["messages"].([]any)
	message := messages[0].(map[string]any)
	contents := message["content"].([]any)

	if len(contents) != 3 {
		t.Fatalf("expected 2 images and 1 text content, got %d", len(contents))
	}

	assertContentHasPrefix(t, contents[0], "image", "data:image/png;base64,")
	assertContentHasPrefix(t, contents[1], "image", "data:image/png;base64,")
	assertContentString(t, contents[2], "text", "blend two images")
}

func TestAdaptorConvertRequestQwenImageEditBaseModelFiltersUnsupportedParams(t *testing.T) {
	adaptor := &Adaptor{}
	m := meta.NewMeta(nil, mode.ImagesEdits, "qwen-image-edit", coremodel.ModelConfig{})
	m.ActualModel = "qwen-image-edit"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("prompt", "add a hat")
	_ = writer.WriteField("size", "1024x1024")
	_ = writer.WriteField("prompt_extend", "true")

	part, err := writer.CreateFormFile("image", "input.png")
	if err != nil {
		t.Fatalf("failed to create file part: %v", err)
	}

	_, _ = part.Write([]byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	})

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/edits",
		body,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	result, err := adaptor.ConvertRequest(m, nil, req)
	if err != nil {
		t.Fatalf("ConvertRequest returned error: %v", err)
	}

	convertedBody, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("failed to read converted body: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(convertedBody, &payload); err != nil {
		t.Fatalf("failed to unmarshal converted body %s: %v", string(convertedBody), err)
	}

	if parameters, ok := payload["parameters"].(map[string]any); ok {
		if _, ok := parameters["size"]; ok {
			t.Fatalf("expected size not to be forwarded for qwen-image-edit, got %#v", parameters)
		}

		if _, ok := parameters["prompt_extend"]; ok {
			t.Fatalf(
				"expected prompt_extend not to be forwarded for qwen-image-edit, got %#v",
				parameters,
			)
		}
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
					"text": "describe this image"
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
	assertContentString(t, contents[1], "text", "describe this image")
	assertContentString(t, contents[2], "text", "plain text")
	assertContentString(t, contents[3], "image", "https://example.com/image.jpg")
}

func TestAdaptorDoResponseQwenImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &Adaptor{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	m := meta.NewMeta(nil, mode.ImagesGenerations, "qwen-image-2.0-pro", coremodel.ModelConfig{})
	m.Set(MetaResponseFormat, "url")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"request_id": "req-1",
			"output": {
				"choices": [
					{
						"message": {
							"content": [
								{"image": "https://example.com/out.png"}
							]
						}
					}
				]
			},
			"usage": {
				"height": 2048,
				"image_count": 1,
				"width": 2048
			}
		}`)),
	}

	result, adaptorErr := adaptor.DoResponse(m, nil, ctx, resp)
	if adaptorErr != nil {
		t.Fatalf("DoResponse returned error: %v", adaptorErr)
	}

	var imageResponse relaymodel.ImageResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &imageResponse); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(imageResponse.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imageResponse.Data))
	}

	if imageResponse.Data[0].URL != "https://example.com/out.png" {
		t.Fatalf("expected image URL, got %#v", imageResponse.Data[0].URL)
	}

	if int64(result.Usage.InputTokens) != 0 ||
		int64(result.Usage.OutputTokens) != 1 ||
		int64(result.Usage.ImageOutputTokens) != 1 ||
		int64(result.Usage.TotalTokens) != 1 {
		t.Fatalf("unexpected usage: %#v", result.Usage)
	}
}

func TestAdaptorDoResponseQwenImageB64DownloadFailureFallsBackToURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	imageServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		}),
	)
	defer imageServer.Close()

	adaptor := &Adaptor{}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	m := meta.NewMeta(nil, mode.ImagesGenerations, "qwen-image-2.0-pro", coremodel.ModelConfig{})
	m.Set(MetaResponseFormat, "b64_json")

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"request_id": "req-1",
			"output": {
				"choices": [
					{
						"message": {
							"content": [
								{"image": "` + imageServer.URL + `/missing.png"}
							]
						}
					}
				]
			},
			"usage": {
				"image_count": 1
			}
		}`)),
	}

	result, adaptorErr := adaptor.DoResponse(m, nil, ctx, resp)
	if adaptorErr != nil {
		t.Fatalf("DoResponse returned error: %v", adaptorErr)
	}

	var imageResponse relaymodel.ImageResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &imageResponse); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(imageResponse.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(imageResponse.Data))
	}

	if imageResponse.Data[0].URL != imageServer.URL+"/missing.png" {
		t.Fatalf("expected fallback image URL, got %#v", imageResponse.Data[0].URL)
	}

	if imageResponse.Data[0].B64Json != "" {
		t.Fatalf(
			"expected empty b64_json on download failure, got %#v",
			imageResponse.Data[0].B64Json,
		)
	}

	if int64(result.Usage.OutputTokens) != 1 ||
		int64(result.Usage.ImageOutputTokens) != 1 ||
		int64(result.Usage.TotalTokens) != 1 {
		t.Fatalf("unexpected usage: %#v", result.Usage)
	}
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

func assertContentHasPrefix(t *testing.T, got any, key, prefix string) {
	t.Helper()

	gotMap, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected content object, got %T", got)
	}

	if len(gotMap) != 1 {
		t.Fatalf("expected content to have 1 key, got %#v", gotMap)
	}

	value, ok := gotMap[key].(string)
	if !ok {
		t.Fatalf("expected %s string, got %#v", key, gotMap[key])
	}

	if !strings.HasPrefix(value, prefix) {
		t.Fatalf("expected %s prefix %q, got %#v", key, prefix, value)
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
