//nolint:testpackage
package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRequestModelVideoGenerationsJobsMultipart(t *testing.T) {
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

	modelName, err := getRequestModel(ctx, mode.VideoGenerationsJobs, "group-1", 7)
	require.NoError(t, err)
	assert.Equal(t, "wan2.7-i2v", modelName)
}

func TestGetRequestModelVideoGenerationsJobsJSON(t *testing.T) {
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

	modelName, err := getRequestModel(ctx, mode.VideoGenerationsJobs, "group-1", 7)
	require.NoError(t, err)
	assert.Equal(t, "wan2.5-t2v-preview", modelName)
}

func TestGetRequestModelVideosMultipart(t *testing.T) {
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

	modelName, err := getRequestModel(ctx, mode.Videos, "group-1", 7)
	require.NoError(t, err)
	assert.Equal(t, "sora-2", modelName)
}

func TestGetRequestModelVideosMultipartWithoutContentLength(t *testing.T) {
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
	req.ContentLength = -1

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	modelName, err := getRequestModel(ctx, mode.Videos, "group-1", 7)
	require.NoError(t, err)
	assert.Equal(t, "sora-2", modelName)
	require.NotNil(t, req.MultipartForm)
}

func TestGetRequestModelVideosJSON(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/videos",
		bytes.NewBufferString(`{"model":"sora-2","prompt":"A city street"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	modelName, err := getRequestModel(ctx, mode.Videos, "group-1", 7)
	require.NoError(t, err)
	assert.Equal(t, "sora-2", modelName)
}
