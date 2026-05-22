package gemini_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/gemini"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/stretchr/testify/assert"
)

func TestConvertImageRequestMapsOpenAIImageToGemini(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: model.ChannelTypeGoogleGemini}
	meta := meta.NewMeta(
		channel,
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
	)

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://localhost/v1/images/generations",
		bytes.NewBufferString(
			`{"model":"gemini-3-pro-image-preview","prompt":"Draw a cat.","size":"1536x1024","response_format":"url"}`,
		),
	)
	assert.NoError(t, err)

	result, err := gemini.ConvertImageRequest(meta, req)
	assert.NoError(t, err)
	assert.Equal(t, "url", meta.GetString(openai.MetaResponseFormat))

	bodyBytes, err := io.ReadAll(result.Body)
	assert.NoError(t, err)

	var geminiReq relaymodel.GeminiChatRequest

	err = json.Unmarshal(bodyBytes, &geminiReq)
	assert.NoError(t, err)
	assert.Len(t, geminiReq.Contents, 1)
	assert.Equal(t, "Draw a cat.", geminiReq.Contents[0].Parts[0].Text)
	assert.Equal(
		t,
		[]string{relaymodel.GeminiModalityText, relaymodel.GeminiModalityImage},
		geminiReq.GenerationConfig.ResponseModalities,
	)
	assert.Equal(t, "3:2", geminiReq.GenerationConfig.ImageConfig.AspectRatio)
}

func TestConvertImageRequestMapsImageSizePreset(t *testing.T) {
	t.Parallel()

	meta := meta.NewMeta(
		&model.Channel{Type: model.ChannelTypeGoogleGemini},
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
	)

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://localhost/v1/images/generations",
		bytes.NewBufferString(
			`{"model":"gemini-3-pro-image-preview","prompt":"Draw a cat.","size":"2K"}`,
		),
	)
	assert.NoError(t, err)

	result, err := gemini.ConvertImageRequest(meta, req)
	assert.NoError(t, err)

	bodyBytes, err := io.ReadAll(result.Body)
	assert.NoError(t, err)

	var geminiReq relaymodel.GeminiChatRequest

	err = json.Unmarshal(bodyBytes, &geminiReq)
	assert.NoError(t, err)
	assert.Equal(t, "2k", geminiReq.GenerationConfig.ImageConfig.ImageSize)
	assert.Empty(t, geminiReq.GenerationConfig.ImageConfig.AspectRatio)
}

func TestConvertImageRequestPreservesStream(t *testing.T) {
	t.Parallel()

	meta := meta.NewMeta(
		&model.Channel{Type: model.ChannelTypeGoogleGemini},
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
	)

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://localhost/v1/images/generations",
		bytes.NewBufferString(
			`{"model":"gemini-3-pro-image-preview","prompt":"Draw a cat.","stream":true}`,
		),
	)
	assert.NoError(t, err)

	_, err = gemini.ConvertImageRequest(meta, req)
	assert.NoError(t, err)
	assert.True(t, meta.GetBool("stream"))

	adaptor := &gemini.Adaptor{}
	requestURL, err := adaptor.GetRequestURL(meta, nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, requestURL.URL, ":streamGenerateContent?alt=sse")
}

func TestConvertImageRequestAllowsMultipleRequestedImages(t *testing.T) {
	t.Parallel()

	meta := meta.NewMeta(
		&model.Channel{Type: model.ChannelTypeGoogleGemini},
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
	)

	req, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"http://localhost/v1/images/generations",
		bytes.NewBufferString(
			`{"model":"gemini-3-pro-image-preview","prompt":"Draw a cat.","n":2}`,
		),
	)
	assert.NoError(t, err)

	result, err := gemini.ConvertImageRequest(meta, req)
	assert.NoError(t, err)

	bodyBytes, err := io.ReadAll(result.Body)
	assert.NoError(t, err)

	var geminiReq relaymodel.GeminiChatRequest

	err = json.Unmarshal(bodyBytes, &geminiReq)
	assert.NoError(t, err)
	assert.Zero(t, geminiReq.GenerationConfig.CandidateCount)
}

func TestGeminiImageAspectRatioFromSize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "1:1", gemini.GeminiImageAspectRatioFromSizeForTest("1024x1024"))
	assert.Equal(t, "3:2", gemini.GeminiImageAspectRatioFromSizeForTest("1536x1024"))
	assert.Equal(t, "2:3", gemini.GeminiImageAspectRatioFromSizeForTest("1024x1536"))
}

func TestImageHandlerConvertsGeminiImageResponseToOpenAI(t *testing.T) {
	t.Parallel()

	meta := meta.NewMeta(
		&model.Channel{Type: model.ChannelTypeGoogleGemini},
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
		meta.WithRequestUsage(model.Usage{
			InputTokens:  model.ZeroNullInt64(1),
			OutputTokens: model.ZeroNullInt64(1),
			TotalTokens:  model.ZeroNullInt64(2),
		}),
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(bytes.NewBufferString(`{
			"candidates":[{
				"content":{"parts":[
					{"text":"done"},
					{"inlineData":{"mimeType":"image/png","data":"aW1hZ2U="}}
				]}
			}],
			"usageMetadata":{
				"promptTokenCount":10,
				"candidatesTokenCount":20,
				"totalTokenCount":30,
				"promptTokensDetails":[{"modality":"IMAGE","tokenCount":3}],
				"candidatesTokensDetails":[{"modality":"IMAGE","tokenCount":20}]
			}
		}`)),
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/images/generations",
		nil,
	)

	result, err := gemini.ImageHandler(meta, c, resp)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), int64(result.Usage.InputTokens))
	assert.Equal(t, int64(20), int64(result.Usage.OutputTokens))
	assert.Equal(t, int64(20), int64(result.Usage.ImageOutputTokens))
	assert.Equal(t, int64(30), int64(result.Usage.TotalTokens))

	var imageResp relaymodel.ImageResponse

	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &imageResp))
	assert.Len(t, imageResp.Data, 1)
	assert.Equal(t, "aW1hZ2U=", imageResp.Data[0].B64Json)
	assert.NotNil(t, imageResp.Usage)
	assert.Equal(t, int64(20), imageResp.Usage.OutputTokens)
	assert.Equal(t, int64(20), imageResp.Usage.OutputTokensDetails.ImageTokens)
}

func TestImageHandlerChargesActualGeminiImageCount(t *testing.T) {
	t.Parallel()

	meta := meta.NewMeta(
		&model.Channel{Type: model.ChannelTypeGoogleGemini},
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
		meta.WithRequestUsage(model.Usage{
			InputTokens:  model.ZeroNullInt64(5),
			OutputTokens: model.ZeroNullInt64(2),
			TotalTokens:  model.ZeroNullInt64(7),
		}),
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(bytes.NewBufferString(`{
			"candidates":[{
				"content":{"parts":[
					{"inlineData":{"mimeType":"image/png","data":"aW1hZ2U="}}
				]}
			}]
		}`)),
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/images/generations",
		nil,
	)

	result, err := gemini.ImageHandler(meta, c, resp)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), int64(result.Usage.OutputTokens))
	assert.Equal(t, int64(1), int64(result.Usage.ImageOutputTokens))
	assert.Equal(t, int64(6), int64(result.Usage.TotalTokens))

	var imageResp relaymodel.ImageResponse

	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &imageResp))
	assert.NotNil(t, imageResp.Usage)
	assert.Equal(t, int64(1), imageResp.Usage.OutputTokens)
	assert.Equal(t, int64(1), imageResp.Usage.OutputTokensDetails.ImageTokens)
}

func TestImageStreamHandlerConvertsGeminiStreamToOpenAIImageEvents(t *testing.T) {
	t.Parallel()

	meta := meta.NewMeta(
		&model.Channel{Type: model.ChannelTypeGoogleGemini},
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		model.ModelConfig{Type: mode.GeminiImage},
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/event-stream"}},
		Body: io.NopCloser(bytes.NewBufferString(
			"data: {\"candidates\":[{\"content\":{\"parts\":[{\"inlineData\":{\"mimeType\":\"image/png\",\"data\":\"cGFydGlhbA==\"}}]}}]}\n\n" +
				"data: {\"usageMetadata\":{\"promptTokenCount\":4,\"candidatesTokenCount\":8,\"totalTokenCount\":12,\"promptTokensDetails\":[],\"candidatesTokensDetails\":[{\"modality\":\"IMAGE\",\"tokenCount\":8}]}}\n\n",
		)),
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/images/generations",
		nil,
	)

	result, err := gemini.ImageHandler(meta, c, resp)
	assert.Nil(t, err)
	assert.Equal(t, int64(8), int64(result.Usage.ImageOutputTokens))

	body := recorder.Body.String()
	assert.Contains(t, body, "event: "+relaymodel.ImageStreamEventPartialImage+"\n")
	assert.Contains(t, body, `"type":"`+relaymodel.ImageStreamEventPartialImage+`"`)
	assert.Contains(t, body, `"partial_image_index":0`)
	assert.Contains(t, body, `"b64_json":"cGFydGlhbA=="`)
	assert.Contains(t, body, "event: "+relaymodel.ImageStreamEventCompleted+"\n")
	assert.Contains(t, body, `"type":"`+relaymodel.ImageStreamEventCompleted+`"`)
	assert.Contains(t, body, `"b64_json":"cGFydGlhbA=="`)
	assert.NotContains(t, body, `[DONE]`)
}
