package gemini_test

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
	adaptorapi "github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/gemini"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/stretchr/testify/require"
)

type geminiVideoTestStore struct {
	saved []adaptorapi.StoreCache
}

func (s *geminiVideoTestStore) GetStore(
	_ string,
	_ int,
	id string,
) (adaptorapi.StoreCache, error) {
	for _, cache := range s.saved {
		if cache.ID == id {
			return cache, nil
		}
	}

	return adaptorapi.StoreCache{}, coremodel.NotFoundError(coremodel.ErrStoreNotFound)
}

func (s *geminiVideoTestStore) SaveStore(cache adaptorapi.StoreCache) error {
	s.saved = append(s.saved, cache)
	return nil
}

func (s *geminiVideoTestStore) SaveStoreWithOption(
	cache adaptorapi.StoreCache,
	_ adaptorapi.SaveStoreOption,
) error {
	s.saved = append(s.saved, cache)
	return nil
}

func (s *geminiVideoTestStore) SaveIfNotExistStore(cache adaptorapi.StoreCache) error {
	s.saved = append(s.saved, cache)
	return nil
}

func TestGetRequestURLGeminiVideoUsesPredictLongRunning(t *testing.T) {
	adaptor := &gemini.Adaptor{}
	m := meta.NewMeta(nil, mode.GeminiVideo, "veo-3.1-generate-preview", coremodel.ModelConfig{})
	m.Channel.BaseURL = "https://generativelanguage.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(m, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, reqURL.Method)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		reqURL.URL,
	)
}

func TestGetRequestURLGeminiVideoOperationsUsesGet(t *testing.T) {
	adaptor := &gemini.Adaptor{}
	m := meta.NewMeta(
		nil,
		mode.GeminiVideoOperations,
		"public-veo",
		coremodel.ModelConfig{},
		meta.WithOperationID("video-123"),
	)
	m.ActualModel = "veo-3.1-generate-preview"
	m.OriginModel = "public-veo"
	m.Channel.BaseURL = "https://generativelanguage.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(m, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.MethodGet, reqURL.Method)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/models/veo-3.1-generate-preview/operations/video-123",
		reqURL.URL,
	)
}

func TestGetRequestURLGeminiVideoOperationsUsesStoredUpstreamOperation(t *testing.T) {
	adaptor := &gemini.Adaptor{}
	store := &geminiVideoTestStore{
		saved: []adaptorapi.StoreCache{
			{
				ID:       coremodel.VideoJobStoreID("video-123"),
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
	m.Channel.BaseURL = "https://generativelanguage.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(m, store, nil)
	require.NoError(t, err)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/video-123",
		reqURL.URL,
	)
}

func TestGetRequestURLVideoGenerationContentUsesOperationID(t *testing.T) {
	adaptor := &gemini.Adaptor{}
	operationName := "models/veo-3.1-generate-preview/operations/video-123"
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsContent,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
		meta.WithGenerationID(operationName+":1"),
	)
	m.Channel.BaseURL = "https://generativelanguage.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(m, nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.MethodGet, reqURL.Method)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/models/veo-3.1-generate-preview/operations/video-123",
		reqURL.URL,
	)
}

func TestGetRequestURLVideoGenerationGetJobsDecodesLocalID(t *testing.T) {
	adaptor := &gemini.Adaptor{}
	operationName := "models/veo-3.1-generate-preview/operations/video-123"
	localID := gemini.GeminiVideoLocalIDForTest(operationName)
	store := &geminiVideoTestStore{
		saved: []adaptorapi.StoreCache{
			{
				ID:       coremodel.VideoJobStoreID(localID),
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
	m.Channel.BaseURL = "https://generativelanguage.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(m, store, nil)
	require.NoError(t, err)
	require.Equal(t, http.MethodGet, reqURL.Method)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/models/veo-3.1-generate-preview/operations/video-123",
		reqURL.URL,
	)
	require.NotContains(t, localID, "/")
}

func TestGetRequestURLVideoGenerationContentDecodesLocalGenerationID(t *testing.T) {
	adaptor := &gemini.Adaptor{}
	operationName := "models/veo-3.1-generate-preview/operations/video-123"
	localGenerationID := gemini.GeminiVideoLocalIDForTest(operationName) + ":1"
	store := &geminiVideoTestStore{
		saved: []adaptorapi.StoreCache{
			{
				ID: coremodel.VideoGenerationStoreID(
					strings.TrimSuffix(localGenerationID, ":1"),
				),
				Metadata: operationName,
			},
		},
	}
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsContent,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
		meta.WithGroup(coremodel.GroupCache{ID: "group-1"}),
		meta.WithToken(coremodel.TokenCache{ID: 7}),
		meta.WithGenerationID(localGenerationID),
	)
	m.Channel.BaseURL = "https://generativelanguage.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(m, store, nil)
	require.NoError(t, err)
	require.Equal(t, http.MethodGet, reqURL.Method)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/models/veo-3.1-generate-preview/operations/video-123",
		reqURL.URL,
	)
}

func TestConvertVideoRequestMapsOpenAIFields(t *testing.T) {
	m := meta.NewMeta(nil, mode.Videos, "veo-3.1-generate-preview", coremodel.ModelConfig{})
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/videos",
		strings.NewReader(
			`{"model":"veo","prompt":"make a video","seconds":8,"n_variants":2,"size":"1280x720","input_reference":"gs://bucket/image.png"}`,
		),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := gemini.ConvertVideoRequestForTest(m, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video","image":{"fileData":{"fileUri":"gs://bucket/image.png"}}}],
			"parameters":{"aspectRatio":"16:9","durationSeconds":8,"numberOfVideos":1}
		}`,
		string(body),
	)
	require.Equal(t, coremodel.ZeroNullInt64(8), m.RequestUsage.OutputTokens)
}

func TestConvertVideoGenerationJobRequestUsesJobOnlyFields(t *testing.T) {
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
		strings.NewReader(
			`{"model":"veo","prompt":"make a video","n_seconds":8,"n_variants":2,"size":"1280x720"}`,
		),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := gemini.ConvertVideoRequestForTest(m, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video"}],
			"parameters":{"aspectRatio":"16:9","durationSeconds":8,"numberOfVideos":2}
		}`,
		string(body),
	)
	require.Equal(t, coremodel.ZeroNullInt64(16), m.RequestUsage.OutputTokens)
}

func TestConvertVideoGenerationJobRequestIgnoresVideosSeconds(t *testing.T) {
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
		strings.NewReader(
			`{"model":"veo","prompt":"make a video","seconds":8,"n_variants":2,"size":"1280x720"}`,
		),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := gemini.ConvertVideoRequestForTest(m, req)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video"}],
			"parameters":{"aspectRatio":"16:9","numberOfVideos":2}
		}`,
		string(body),
	)
}

func TestVideoSubmitHandlerReturnsURLSafeJobID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &gemini.Adaptor{}
	store := &geminiVideoTestStore{}
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsJobs,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
	)
	m.OriginModel = "veo-3.1-generate-preview"
	m.Group.ID = "group-1"
	m.Token.ID = 7
	m.Channel.ID = 11

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		nil,
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"name":"models/veo-3.1-generate-preview/operations/video-123","done":false}`,
		)),
	}

	result, relayErr := adaptor.DoResponse(m, store, c, resp)
	require.Nil(t, relayErr)
	require.True(t, result.AsyncUsage)
	require.Equal(t, "models/veo-3.1-generate-preview/operations/video-123", result.UpstreamID)

	var job relaymodel.VideoGenerationJob
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &job))
	require.NotEmpty(t, job.ID)
	require.NotContains(t, job.ID, "/")
	require.Equal(t, coremodel.VideoJobStoreID(job.ID), store.saved[0].ID)
	require.Equal(t, result.UpstreamID, store.saved[0].Metadata)
	require.LessOrEqual(t, len(store.saved[0].ID), 128)
}

func TestNativeVideoHandlerStoresShortOperationID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &gemini.Adaptor{}
	store := &geminiVideoTestStore{}
	m := meta.NewMeta(
		nil,
		mode.GeminiVideo,
		"public-veo",
		coremodel.ModelConfig{},
	)
	m.ActualModel = "veo-3.1-generate-preview"
	m.OriginModel = "public-veo"
	m.Group.ID = "group-1"
	m.Token.ID = 7
	m.Channel.ID = 11

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1beta/models/veo-3.1-generate-preview:predictLongRunning",
		nil,
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"name":"models/veo-3.1-generate-preview/operations/video-123","done":false}`,
		)),
	}

	result, relayErr := adaptor.DoResponse(m, store, c, resp)
	require.Nil(t, relayErr)
	require.True(t, result.AsyncUsage)
	require.Equal(t, "models/veo-3.1-generate-preview/operations/video-123", result.UpstreamID)
	require.Equal(t, coremodel.VideoJobStoreID("video-123"), store.saved[0].ID)
	require.Equal(t, result.UpstreamID, store.saved[0].Metadata)
	require.JSONEq(
		t,
		`{"name":"models/public-veo/operations/video-123","done":false}`,
		recorder.Body.String(),
	)
}

func TestVideoSubmitHandlerStoresBoundedIDForLongVertexOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &gemini.Adaptor{}
	store := &geminiVideoTestStore{}
	m := meta.NewMeta(
		nil,
		mode.VideoGenerationsJobs,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
	)
	m.OriginModel = "veo-3.1-generate-preview"
	m.Group.ID = "group-1"
	m.Token.ID = 7
	m.Channel.ID = 11
	m.Channel.BaseURL = "https://us-central1-aiplatform.googleapis.com"

	operationName := "projects/project-1/locations/us-central1/publishers/google/models/veo-3.1-generate-preview/operations/1234567890abcdef1234567890abcdef"

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/v1/video/generations/jobs",
		nil,
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"name":"` + operationName + `","done":false}`,
		)),
	}

	_, relayErr := adaptor.DoResponse(m, store, c, resp)
	require.Nil(t, relayErr)

	var job relaymodel.VideoGenerationJob
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &job))
	require.NotContains(t, job.ID, "/")
	require.LessOrEqual(t, len(coremodel.VideoJobStoreID(job.ID)), 128)
	require.Equal(t, operationName, store.saved[0].Metadata)

	followupMeta := meta.NewMeta(
		nil,
		mode.VideoGenerationsGetJobs,
		"veo-3.1-generate-preview",
		coremodel.ModelConfig{},
		meta.WithGroup(m.Group),
		meta.WithToken(m.Token),
		meta.WithJobID(job.ID),
	)
	followupMeta.Channel.BaseURL = "https://us-central1-aiplatform.googleapis.com"

	reqURL, err := adaptor.GetRequestURL(followupMeta, store, nil)
	require.NoError(t, err)
	require.Equal(
		t,
		"https://us-central1-aiplatform.googleapis.com/v1beta/"+operationName,
		reqURL.URL,
	)
}

func TestVideoStatusHandlerResolvesStoredOperationWhenResponseNameMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adaptor := &gemini.Adaptor{}
	operationName := "models/veo-3.1-generate-preview/operations/video-123"
	localID := gemini.GeminiVideoLocalIDForTest(operationName)
	store := &geminiVideoTestStore{
		saved: []adaptorapi.StoreCache{
			{
				ID:       coremodel.VideoJobStoreID(localID),
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
	m.OriginModel = "veo-3.1-generate-preview"

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/v1/video/generations/jobs/"+localID,
		nil,
	)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(
			`{"done":true,"response":{"generateVideoResponse":{"generatedSamples":[{"video":{"uri":"https://example.com/video.mp4"}}]}}}`,
		)),
	}

	_, relayErr := adaptor.DoResponse(m, store, c, resp)
	require.Nil(t, relayErr)

	var job relaymodel.VideoGenerationJob
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &job))
	require.Equal(t, localID, job.ID)
	require.Len(t, job.Generations, 1)
	require.Equal(t, localID, job.Generations[0].ID)
	require.Equal(t, operationName, store.saved[1].Metadata)
}

func TestConvertVideoRequestInjectsPersonGenerationWhenEnabled(t *testing.T) {
	m := meta.NewMeta(nil, mode.Videos, "veo-3.1-generate-preview", coremodel.ModelConfig{})
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/videos",
		strings.NewReader(`{"model":"veo","prompt":"make a video"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := gemini.ConvertVideoRequestWithConfigForTest(
		m,
		req,
		gemini.Config{EnablePersonGenerationAllowAll: true},
	)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video"}],
			"parameters":{"numberOfVideos":1,"personGeneration":"allow_all"}
		}`,
		string(body),
	)
}

func TestConvertVideoRequestKeepsRequestPersonGeneration(t *testing.T) {
	m := meta.NewMeta(nil, mode.Videos, "veo-3.1-generate-preview", coremodel.ModelConfig{})
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/videos",
		strings.NewReader(
			`{"model":"veo","prompt":"make a video","person_generation":"allow_adult"}`,
		),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	result, err := gemini.ConvertVideoRequestWithConfigForTest(
		m,
		req,
		gemini.Config{EnablePersonGenerationAllowAll: true},
	)
	require.NoError(t, err)

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.JSONEq(
		t,
		`{
			"instances":[{"prompt":"make a video"}],
			"parameters":{"numberOfVideos":1,"personGeneration":"allow_adult"}
		}`,
		string(body),
	)
}

func TestGeminiVideoURLByIDSelectsVariantSuffix(t *testing.T) {
	operationName := "models/veo-3.1-generate-preview/operations/video-123"
	operation := &relaymodel.GeminiVideoOperation{
		Response: relaymodel.GeminiVideoOperationResponse{
			GenerateVideoResponse: relaymodel.GeminiGenerateVideoResponse{
				GeneratedSamples: []relaymodel.GeminiGeneratedSample{
					{Video: relaymodel.GeminiGeneratedVideo{URI: "https://example.com/first.mp4"}},
					{Video: relaymodel.GeminiGeneratedVideo{URI: "https://example.com/second.mp4"}},
				},
			},
		},
	}

	require.Equal(
		t,
		"https://example.com/first.mp4",
		gemini.GeminiVideoURLByIDForTest(operation, operationName),
	)
	require.Equal(
		t,
		"https://example.com/second.mp4",
		gemini.GeminiVideoURLByIDForTest(operation, operationName+":1"),
	)
	require.Empty(t, gemini.GeminiVideoURLByIDForTest(operation, operationName+":9"))
}

func TestGeminiVideoOperationResponseExtractsGeneratedSampleURI(t *testing.T) {
	body := `{
		"name": "models/veo-3.1-generate-preview/operations/9p5ug42emn2z",
		"done": true,
		"response": {
			"@type": "type.googleapis.com/google.ai.generativelanguage.v1beta.PredictLongRunningResponse",
			"generateVideoResponse": {
				"generatedSamples": [
					{
						"video": {
							"uri": "https://generativelanguage.googleapis.com/v1beta/files/04pmwg9vlfwb:download?alt=media"
						}
					}
				]
			}
		}
	}`

	var operation relaymodel.GeminiVideoOperation
	require.NoError(t, json.Unmarshal([]byte(body), &operation))

	require.Equal(t, "models/veo-3.1-generate-preview/operations/9p5ug42emn2z", operation.Name)
	require.True(t, operation.Done)
	require.Equal(
		t,
		"https://generativelanguage.googleapis.com/v1beta/files/04pmwg9vlfwb:download?alt=media",
		gemini.GeminiVideoURLByIDForTest(&operation, operation.Name),
	)
}

func TestFetchAsyncUsageKeepsRequestUsageAndResolution(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(
			t,
			"/v1beta/models/veo-3.1-generate-preview/operations/video-123",
			r.URL.Path,
		)
		require.Equal(t, "apikey", r.Header.Get("X-Goog-Api-Key"))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"name":"models/veo-3.1-generate-preview/operations/video-123",
			"done":true,
			"response":{
				"generateVideoResponse":{
					"generatedSamples":[{"video":{"uri":"https://example.com/video.mp4"}}]
				},
				"usageMetadata":{
					"promptTokenCount":1,
					"candidatesTokenCount":999,
					"totalTokenCount":1000,
					"promptTokensDetails":[],
					"candidatesTokensDetails":[]
				}
			}
		}`))
	}))
	defer ts.Close()

	adaptor := &gemini.Adaptor{}
	usage, usageContext, done, err := adaptor.FetchAsyncUsage(
		t.Context(),
		adaptorapi.AsyncUsageRequest{
			Channel: &coremodel.Channel{Key: "apikey", BaseURL: ts.URL},
			Info: &coremodel.AsyncUsageInfo{
				Mode:       int(mode.GeminiVideo),
				Model:      "veo-3.1-generate-preview",
				BaseURL:    ts.URL,
				UpstreamID: "models/veo-3.1-generate-preview/operations/video-123",
				Usage: coremodel.Usage{
					OutputTokens: 12,
					TotalTokens:  12,
				},
				UsageContext: coremodel.UsageContext{
					PriceCondition: coremodel.UsagePriceCondition{Resolution: "1080p"},
				},
			},
		},
	)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, coremodel.ZeroNullInt64(12), usage.OutputTokens)
	require.Equal(t, coremodel.ZeroNullInt64(12), usage.TotalTokens)
	require.Equal(t, "1080p", usageContext.PriceCondition.Resolution)
}
