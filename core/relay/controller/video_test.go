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
	require.Equal(t, model.ZeroNullInt64(5), usage.Usage.OutputTokens)
	require.Equal(t, model.ZeroNullInt64(5), usage.Usage.TotalTokens)
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
	require.Equal(t, model.ZeroNullInt64(8), usage.Usage.OutputTokens)
	require.Equal(t, model.ZeroNullInt64(8), usage.Usage.TotalTokens)
}

func TestGetVideoGenerationJobRequestUsageVideosMultipart(t *testing.T) {
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

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Equal(t, model.ZeroNullInt64(6), usage.Usage.OutputTokens)
	require.Equal(t, model.ZeroNullInt64(6), usage.Usage.TotalTokens)
}

func TestGetVideoGenerationJobRequestUsageSetsResolutionCondition(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	body := `{
		"model":"video-model",
		"prompt":"A city street",
		"width":1280,
		"height":720,
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

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Equal(t, model.ZeroNullInt64(5), usage.Usage.OutputTokens)
	require.Equal(t, "720p", usage.Context.PriceCondition.Size)
}

func TestGetVideoGenerationJobRequestUsageRejectsUnsupportedSize(t *testing.T) {
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
			model.ModelConfigVideoSizes: []string{"1080p"},
		},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported video size `720p`", err.Error())
}

func TestGetVideoGenerationJobRequestUsageRejectsTooLongSeconds(t *testing.T) {
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
}

func TestGetVideoGenerationJobRequestUsageRejectsNegativeSeconds(t *testing.T) {
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

	_, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.Error(t, err)
	require.Equal(t, "invalid seconds: must be non-negative", err.Error())
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
					Condition: model.PriceCondition{Size: "1280x720"},
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
