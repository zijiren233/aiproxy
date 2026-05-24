//nolint:testpackage
package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
)

func TestValidateImagesRequestSkipsMissingN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/images/generations",
		strings.NewReader(`{"model":"gpt-image-1","prompt":"test"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{MaxImageGenerationCount: 1})
	require.NoError(t, err)
}

func TestValidateImagesRequestRejectsTooLargeN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/images/generations",
		strings.NewReader(`{"model":"gpt-image-1","prompt":"test","n":2}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{MaxImageGenerationCount: 1})
	require.Error(t, err)
	require.Equal(t, "n must be less than or equal to 1", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestValidateImagesRequestRejectsDuplicateN(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/images/generations",
		strings.NewReader(`{"model":"gpt-image-1","prompt":"test","n":1,"n":100}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{MaxImageGenerationCount: 1})
	require.Error(t, err)
	require.Equal(t, "duplicate n", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestGetImagesRequestPriceUsesPerImageUnitForConditionalPrices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1024x1024"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	price, err := GetImagesRequestPrice(c, model.ModelConfig{
		Price: model.Price{
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"1024x1024"}},
					Price:     model.Price{OutputPrice: 0.12},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, price.ConditionalPrices, 1)
	require.Equal(t, model.ZeroNullInt64(1), price.OutputPriceUnit)
	require.Equal(t, model.ZeroNullFloat64(0.12), price.ConditionalPrices[0].Price.OutputPrice)
	require.Equal(t, model.ZeroNullInt64(1), price.ConditionalPrices[0].Price.OutputPriceUnit)
}

func TestGetImagesRequestPriceUsesPerImageOutputUnitForConditionalPrices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1024x1024"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	price, err := GetImagesRequestPrice(c, model.ModelConfig{
		Price: model.Price{
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"1024x1024"}},
					Price:     model.Price{ImageOutputPrice: 0.12},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, price.ConditionalPrices, 1)
	require.Equal(t, model.ZeroNullInt64(1), price.ImageOutputPriceUnit)
	require.Equal(
		t,
		model.ZeroNullFloat64(0.12),
		price.ConditionalPrices[0].Price.ImageOutputPrice,
	)
	require.Equal(t, model.ZeroNullInt64(1), price.ConditionalPrices[0].Price.ImageOutputPriceUnit)
}

func TestGetImagesRequestPriceAllowsUnmatchedConditionalPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"512x512"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	price, err := GetImagesRequestPrice(c, model.ModelConfig{
		Price: model.Price{
			ConditionalPrices: []model.ConditionalPrice{
				{
					Condition: model.PriceCondition{Resolution: []string{"1024x1024"}},
					Price:     model.Price{OutputPrice: 0.12},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, price.ConditionalPrices, 1)
	require.Equal(t, model.ZeroNullInt64(1), price.OutputPriceUnit)
}

func TestValidateImagesRequestRejectsUnsupportedResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"512x512"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{
		AllowedResolutions: []string{"1024x1024"},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported image resolution `512x512`", err.Error())

	var requestParamErr *RequestParamError
	require.ErrorAs(t, err, &requestParamErr)
	require.Equal(t, 400, requestParamErr.StatusCode)
}

func TestValidateImagesRequestMatchesGeminiStyleImageResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1024x1024"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{
		AllowedResolutions: []string{"1k"},
	})
	require.NoError(t, err)
}

func TestValidateImagesRequestDisableFuzzyRejectsGeminiStyleImageResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1024x1024"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{
		AllowedResolutions:          []string{"1k"},
		DisableResolutionFuzzyMatch: true,
	})
	require.Error(t, err)
	require.Equal(t, "unsupported image resolution `1024x1024`", err.Error())
}

func TestValidateImagesRequestDoesNotMatchAspectRatioAsResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1024x1024"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{
		AllowedResolutions: []string{"1:1"},
	})
	require.Error(t, err)
	require.Equal(t, "unsupported image resolution `1024x1024`", err.Error())
}

func TestValidateImagesRequestRejectsAspectRatioRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1:1"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{
		AllowedResolutions: []string{"1:1"},
	})
	require.Error(t, err)
	require.Equal(t, "invalid image resolution `1:1`", err.Error())
}

func TestValidateImagesRequestRejectsInvalidResolutionFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1k"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{})
	require.Error(t, err)
	require.Equal(t, "invalid image resolution `1k`", err.Error())
}

func TestValidateImagesRequestRejectsNonOpenAIDimensionDelimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(`{"model":"image-model","prompt":"A city street","size":"1024*1024"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	err := ValidateImagesRequest(c, model.ModelConfig{})
	require.Error(t, err)
	require.Equal(t, "invalid image resolution `1024*1024`", err.Error())
}

func TestGetImagesRequestUsageSetsPriceCondition(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(
			`{"model":"image-model","prompt":"A city street","size":"1024x1024","quality":"hd"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	usage, err := GetImagesRequestUsage(c, model.ModelConfig{})
	require.NoError(t, err)
	require.Equal(t, "1024x1024", usage.Context.Resolution)
	require.Equal(t, "hd", usage.Context.Quality)
	require.Zero(t, usage.Usage.InputTokens)
	require.Zero(t, usage.Usage.ImageInputTokens)
	require.Zero(t, usage.Usage.OutputTokens)
	require.Zero(t, usage.Usage.ImageOutputTokens)
	require.Zero(t, usage.Usage.TotalTokens)
}
