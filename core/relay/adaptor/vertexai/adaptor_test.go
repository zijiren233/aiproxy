package vertexai_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	coremodel "github.com/labring/aiproxy/core/model"
	adaptorapi "github.com/labring/aiproxy/core/relay/adaptor"
	vertexai "github.com/labring/aiproxy/core/relay/adaptor/vertexai"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vertexVideoTestStore struct {
	items map[string]adaptorapi.StoreCache
}

func (s *vertexVideoTestStore) GetStore(
	_ string,
	_ int,
	id string,
) (adaptorapi.StoreCache, error) {
	if item, ok := s.items[id]; ok {
		return item, nil
	}

	return adaptorapi.StoreCache{}, coremodel.NotFoundError(coremodel.ErrStoreNotFound)
}

func (s *vertexVideoTestStore) SaveStore(adaptorapi.StoreCache) error {
	return nil
}

func (s *vertexVideoTestStore) SaveStoreWithOption(
	adaptorapi.StoreCache,
	adaptorapi.SaveStoreOption,
) error {
	return nil
}

func (s *vertexVideoTestStore) SaveIfNotExistStore(adaptorapi.StoreCache) error {
	return nil
}

func TestGetRequestURL_UsesOriginModelNameForRouting(t *testing.T) {
	adaptor := &vertexai.Adaptor{}

	t.Run("origin gemini chooses google publisher endpoint", func(t *testing.T) {
		m := meta.NewMeta(nil, mode.ChatCompletions, "gemini-2.5-pro", coremodel.ModelConfig{})
		m.ActualModel = "mapped-model"
		m.Channel.Key = "us-central1|apikey"
		m.Set("stream", false)

		reqURL, err := adaptor.GetRequestURL(m, nil, nil)
		require.NoError(t, err)
		assert.Equal(
			t,
			http.MethodPost,
			reqURL.Method,
		)
		assert.Contains(
			t,
			reqURL.URL,
			"/publishers/google/models/mapped-model:generateContent",
		)
	})

	t.Run("origin claude chooses anthropic publisher endpoint", func(t *testing.T) {
		m := meta.NewMeta(nil, mode.ChatCompletions, "claude-sonnet-4-5", coremodel.ModelConfig{})
		m.ActualModel = "mapped-model"
		m.Channel.Key = "us-central1|apikey"
		m.Set("stream", true)

		reqURL, err := adaptor.GetRequestURL(m, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, http.MethodPost, reqURL.Method)
		assert.Contains(
			t,
			reqURL.URL,
			"/publishers/anthropic/models/mapped-model:streamRawPredict?alt=sse",
		)
	})
}

func TestGetRequestURLGeminiVideoUsesPredictLongRunning(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	m := meta.NewMeta(nil, mode.GeminiVideo, "veo-3.1-generate-preview", coremodel.ModelConfig{})
	m.Channel.Key = "us-central1|project-1|apikey"

	reqURL, err := adaptor.GetRequestURL(m, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, reqURL.Method)
	assert.Contains(
		t,
		reqURL.URL,
		"/projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview:predictLongRunning",
	)
}

func TestGetRequestURLGeminiVideoOperationUsesGet(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.GeminiVideoOperations,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
		meta.WithOperationID("video-123"),
	)
	m.Channel.Key = "us-central1|project-1|apikey"

	reqURL, err := adaptor.GetRequestURL(m, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, reqURL.Method)
	assert.Contains(
		t,
		reqURL.URL,
		"/projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/video-123",
	)
}

func TestGetRequestURLGeminiVideoOperationUsesStoredFullVertexOperation(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	store := &vertexVideoTestStore{
		items: map[string]adaptorapi.StoreCache{
			coremodel.VideoJobStoreID("video-123"): {
				Metadata: "projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/video-123",
			},
		},
	}
	m := meta.NewMeta(
		nil,
		mode.GeminiVideoOperations,
		"public-veo",
		coremodel.ModelConfig{},
		meta.WithGroup(coremodel.GroupCache{ID: "group-1"}),
		meta.WithToken(coremodel.TokenCache{ID: 7}),
		meta.WithOperationID("video-123"),
	)
	m.ActualModel = "veo-3.1-generate-preview"
	m.OriginModel = "public-veo"
	m.Channel.Key = "us-central1|project-1|apikey"

	reqURL, err := adaptor.GetRequestURL(m, store, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, reqURL.Method)
	assert.Equal(
		t,
		"https://us-central1-aiplatform.googleapis.com/v1/projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/video-123",
		reqURL.URL,
	)
}

func TestGetRequestURLGeminiVideoGenerationContentUsesOperationID(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	operationName := "models/veo-3.1-generate-preview/operations/video-123"
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsContent,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
		meta.WithGenerationID(operationName+":1"),
	)
	m.Channel.Key = "us-central1|project-1|apikey"

	reqURL, err := adaptor.GetRequestURL(m, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, reqURL.Method)
	assert.Contains(
		t,
		reqURL.URL,
		"/projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/video-123",
	)
}

func TestGetRequestURLVideoGenerationGetJobsRestoresLongVertexOperation(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	operationName := "projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/1234567890abcdef1234567890abcdef"
	localID := "gemini_op_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	store := &vertexVideoTestStore{
		items: map[string]adaptorapi.StoreCache{
			coremodel.VideoJobStoreID(localID): {
				ID:       coremodel.VideoJobStoreID(localID),
				Model:    "veo-3.1-generate-preview",
				Metadata: operationName,
			},
		},
	}
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsGetJobs,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
		meta.WithGroup(coremodel.GroupCache{ID: "group-1"}),
		meta.WithToken(coremodel.TokenCache{ID: 7}),
		meta.WithJobID(localID),
	)
	m.Channel.Key = "us-central1|project-1|apikey"

	reqURL, err := adaptor.GetRequestURL(m, store, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, reqURL.Method)
	assert.Equal(
		t,
		"https://us-central1-aiplatform.googleapis.com/v1/"+operationName,
		reqURL.URL,
	)
	assert.LessOrEqual(t, len(coremodel.VideoJobStoreID(localID)), 128)
}

func TestConvertRequestVeoVideoUsesGeminiInnerAdaptor(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsJobs,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
	)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		strings.NewReader(`{"model":"veo","prompt":"make a video","n_seconds":5}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := adaptor.ConvertRequest(m, nil, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video"}],
			"parameters":{"durationSeconds":5,"sampleCount":1}
		}`,
		string(body),
	)
}

func TestConvertRequestNativeGeminiVideoMapsNumberOfVideosToVertexSampleCount(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.GeminiVideo,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
	)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		strings.NewReader(
			`{"instances":[{"prompt":"make a video"}],"parameters":{"durationSeconds":5,"numberOfVideos":2}}`,
		),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := adaptor.ConvertRequest(m, nil, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video"}],
			"parameters":{"durationSeconds":5,"sampleCount":2}
		}`,
		string(body),
	)
}

func TestSetupRequestHeaderVeoVideoUsesGeminiInnerAdaptor(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsJobs,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{Type: mode.GeminiVideo},
	)
	m.Channel.Key = "us-central1|project-1|apikey"

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://example.com/v1/video/generations/jobs",
		nil,
	)
	require.NoError(t, err)

	err = adaptor.SetupRequestHeader(m, nil, nil, req)
	require.NoError(t, err)
	assert.Equal(t, "apikey", req.Header.Get("X-Goog-Api-Key"))
}

func TestConvertRequestGeminiImageUsesGeminiInnerAdaptor(t *testing.T) {
	adaptor := &vertexai.Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.ImagesGenerations,
		"gemini-3-pro-image-preview",
		coremodel.ModelConfig{Type: mode.GeminiImage},
	)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/images/generations",
		strings.NewReader(
			`{"model":"gemini-3-pro-image-preview","prompt":"draw a cat","size":"1024x1024"}`,
		),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := adaptor.ConvertRequest(m, nil, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"contents":[{"role":"user","parts":[{"text":"draw a cat"}]}],
			"generationConfig":{
				"responseModalities":["TEXT","IMAGE"],
				"imageConfig":{"aspectRatio":"1:1"}
			}
		}`,
		string(body),
	)
}
