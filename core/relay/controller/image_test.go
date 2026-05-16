//nolint:testpackage
package controller

import (
	"context"
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
