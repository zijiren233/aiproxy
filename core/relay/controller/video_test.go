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
	require.Zero(t, usage)
}

func TestGetVideoGenerationJobRequestUsageJSON(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		bytes.NewBufferString(`{"model":"wan2.5-t2v-preview","prompt":"A city street"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	usage, err := GetVideoGenerationJobRequestUsage(ctx, model.ModelConfig{})
	require.NoError(t, err)
	require.Zero(t, usage)
}

func TestGetVideoGenerationJobRequestUsageVideosMultipart(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "sora-2"))
	require.NoError(t, writer.WriteField("prompt", "Animate the reference"))
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
	require.Zero(t, usage)
}
