//nolint:testpackage
package controller

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
)

func TestGetVideoGenerationJobRequestUsageMultipart(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "wan2.7-i2v"))
	require.NoError(t, writer.WriteField("prompt", "Animate the reference"))
	require.NoError(t, writer.WriteField("n_seconds", "5"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestGetVideoGenerationJobRequestUsageJSON(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"wan2.5-t2v-preview",
		"prompt":"A city street",
		"n_seconds":4,
		"n_variants":2
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestGetGeminiVideoRequestUsageJSON(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":2}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
	require.Equal(t, "720p", usage.Context.Resolution)
}

func TestGetGeminiVideoRequestUsageReadsTopLevelNativeParameters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"durationSeconds":7,
		"numberOfVideos":3
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestGetGeminiVideoRequestUsageTopLevelZeroFallsBackToParameters(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"durationSeconds":0,
		"numberOfVideos":0,
		"parameters":{"durationSeconds":6,"numberOfVideos":2}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestValidateGeminiVideoRequestRejectsTooLongDuration(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":1}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateGeminiVideoRequest(ctx, model.ModelConfig{
		MaxVideoGenerationSeconds: 5,
	})
	require.Error(t, err)
	require.Equal(t, "seconds must be less than or equal to 5", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestValidateGeminiVideoRequestRejectsTooLongTopLevelDuration(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"durationSeconds":60,
		"numberOfVideos":1
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateGeminiVideoRequest(ctx, model.ModelConfig{
		MaxVideoGenerationSeconds: 15,
	})
	require.Error(t, err)
	require.Equal(t, "seconds must be less than or equal to 15", err.Error())
}

func TestValidateGeminiVideoRequestRejectsTooManyVideos(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":3}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateGeminiVideoRequest(ctx, model.ModelConfig{
		MaxVideoGenerationCount: 2,
	})
	require.Error(t, err)
	require.Equal(t, "video count must be less than or equal to 2", err.Error())
}

func TestValidateGeminiVideoRequestRejectsUnsupportedResolution(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"resolution":"1080p"}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateGeminiVideoRequest(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"720p"},
		},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported video resolution `1080p`", err.Error())
}

func TestGetGeminiVideoRequestUsageSetsResolutionCondition(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":2,"resolution":"1080p"}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"1080p"},
		},
	})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Equal(t, "1080p", usage.Context.Resolution)
}

func TestGetGeminiVideoRequestUsageIgnoresAspectRatioForResolutionCondition(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":2,"aspectRatio":"16:9"}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"720p"},
		},
	})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Equal(t, "720p", usage.Context.Resolution)
}

func TestGetGeminiVideoRequestUsageRejectsWhenDefaultResolutionUnsupported(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"aspectRatio":"1280x720"}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"1080p"},
		},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported video resolution `720p`", err.Error())
}

func TestGetGeminiVideoRequestUsageResolutionOverridesAspectRatio(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"resolution":"1080p","aspectRatio":"16:9"}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"1080p"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "1080p", usage.Context.Resolution)
}

func TestGetGeminiVideoRequestUsageDefaultsWhenParametersMissing(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{"instances":[{"prompt":"A city street"}]}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
	require.Equal(t, "720p", usage.Context.Resolution)
}

func TestGetGeminiVideoRequestUsageIgnoresVertexSampleCount(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"sampleCount":2}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestGetGeminiVideoRequestUsageRejectsUnsupportedResolution(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"resolution":"1080p"}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"720p"},
		},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported video resolution `1080p`", err.Error())
}

func TestValidateVideoGenerationJobRequestRejectsTooManyVariants(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"n_seconds":6,
		"n_variants":3
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateVideoGenerationJobRequest(ctx, model.ModelConfig{
		MaxVideoGenerationCount: 2,
	})
	require.Error(t, err)
	require.Equal(t, "video count must be less than or equal to 2", err.Error())
}

func TestGetGeminiVideoRequestUsageRejectsTooLongDuration(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":1}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{
		MaxVideoGenerationSeconds: 5,
	})
	require.Error(t, err)
	require.Equal(t, "seconds must be less than or equal to 5", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestGetGeminiVideoRequestUsageRejectsNegativeNativeDuration(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":-1}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetGeminiVideoRequestUsage(ctx, model.ModelConfig{})
	require.Error(t, err)
	require.Equal(t, "invalid durationSeconds: must be non-negative", err.Error())
}

func TestGetGeminiVideoRequestPriceDoesNotParseOpenAIVideoFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"instances":[{"prompt":"A city street"}],
		"parameters":{"durationSeconds":6,"numberOfVideos":1}
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	price, err := GetGeminiVideoRequestPrice(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"1080p"},
		},
		Price: model.Price{
			OutputPrice: 0.2,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"1080p"}},
					Price:     model.Price{OutputPrice: 0.5},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, model.ZeroNullFloat64(0.2), price.OutputPrice)
	require.Equal(t, model.ZeroNullInt64(1), price.OutputPriceUnit)
	require.Equal(t, model.ZeroNullInt64(1), price.ConditionalPrices[0].Price.OutputPriceUnit)
}

func TestGetVideosRequestUsageMultipart(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "sora-2"))
	require.NoError(t, writer.WriteField("prompt", "Animate the reference"))
	require.NoError(t, writer.WriteField("seconds", "6"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideosRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestValidateVideoGenerationJobRequestRejectsTooLongNSeconds(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"n_seconds":6
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateVideoGenerationJobRequest(ctx, model.ModelConfig{
		MaxVideoGenerationSeconds: 5,
	})
	require.Error(t, err)
	require.Equal(t, "seconds must be less than or equal to 5", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestValidateVideosRequestRejectsTooLongSeconds(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"seconds":6
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateVideosRequest(ctx, model.ModelConfig{
		MaxVideoGenerationSeconds: 5,
	})
	require.Error(t, err)
	require.Equal(t, "seconds must be less than or equal to 5", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestValidateVideosRequestRejectsUnsupportedMultipartSize(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "video-model"))
	require.NoError(t, writer.WriteField("prompt", "A city street"))
	require.NoError(t, writer.WriteField("size", "1080p"))
	require.NoError(t, writer.WriteField("seconds", "5"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos/video-123/remix",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateVideosRequest(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"720p"},
		},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported video resolution `1080p`", err.Error())
}

func TestValidateVideosRequestIgnoresJobOnlyFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"seconds":4,
		"n_seconds":60,
		"n_variants":10,
		"width":1920,
		"height":1080
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	err := ValidateVideosRequest(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"720p"},
		},
		MaxVideoGenerationSeconds: 5,
	})
	require.NoError(t, err)
}

func TestGetVideosRequestUsageUsesOfficialVideoFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"seconds":4,
		"size":"720p",
		"n_seconds":60,
		"n_variants":10
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideosRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
	require.Equal(t, "720p", usage.Context.Resolution)
}

func TestGetVideoGenerationJobRequestUsageSetsResolutionCondition(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"width":1280,
		"height":720,
		"n_seconds":5
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Equal(t, "720p", usage.Context.Resolution)
}

func TestGetVideoGenerationJobRequestUsageRejectsUnsupportedResolution(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"size":"720p",
		"seconds":5
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"1080p"},
		},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported video resolution `720p`", err.Error())
}

func TestGetVideoGenerationJobRequestUsageNormalizesSupportedResolutionCase(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"size":"720P",
		"seconds":5
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{
		Config: map[model.ModelConfigKey]any{
			model.ModelConfigVideoResolutions: []string{"720p"},
		},
	})
	require.NoError(t, err)
}

func TestGetVideoGenerationJobRequestUsageRejectsTooLongSeconds(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"n_seconds":6
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{
		MaxVideoGenerationSeconds: 5,
	})
	require.Error(t, err)
	require.Equal(t, "seconds must be less than or equal to 5", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestGetVideoGenerationJobRequestUsageRejectsNegativeNSeconds(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"n_seconds":-1,
		"seconds":60
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	_, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.Error(t, err)
	require.Equal(t, "invalid n_seconds: must be non-negative", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestGetVideoGenerationJobRequestUsageIgnoresVideosSecondsField(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"seconds":-1
	}`
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}

func TestGetVideoGenerationJobRequestPriceSetsPerSecondUnitForConditionalPrices(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "video-model"))
	require.NoError(t, writer.WriteField("prompt", "A city street"))
	require.NoError(t, writer.WriteField("size", "1280*720"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		&body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	price, err := GetVideoGenerationJobRequestPrice(ctx, model.ModelConfig{
		Price: model.Price{
			OutputPrice: 0.2,
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"1280x720"}},
					Price:     model.Price{OutputPrice: 0.5},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, model.ZeroNullFloat64(0.2), price.OutputPrice)
	require.Equal(t, model.ZeroNullInt64(1), price.OutputPriceUnit)
	require.Equal(t, model.ZeroNullInt64(1), price.ConditionalPrices[0].Price.OutputPriceUnit)
}
