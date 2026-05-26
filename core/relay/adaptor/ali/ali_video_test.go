//nolint:testpackage
package ali

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
)

func TestConvertAliNativeVideoRequestPreservesBodyAndRewritesModel(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/services/aigc/video-generation/video-synthesis",
		bytes.NewBufferString(
			`{"model":"wan2.5-t2v-preview","input":{"prompt":"go"},"parameters":{"size":"720P"}}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")

	result, err := ConvertAliNativeVideoRequest(&meta.Meta{
		ActualModel: "mapped-wan",
	}, req)
	if err != nil {
		t.Fatalf("ConvertAliNativeVideoRequest returned error: %v", err)
	}

	if got := result.Header.Get("X-Dashscope-Async"); got != "enable" {
		t.Fatalf("expected async header, got %q", got)
	}

	var body map[string]any
	if err := json.NewDecoder(result.Body).Decode(&body); err != nil {
		t.Fatalf("decode converted body: %v", err)
	}

	if body["model"] != "mapped-wan" {
		t.Fatalf("model was not rewritten: %#v", body["model"])
	}

	input, ok := body["input"].(map[string]any)
	if !ok || input["prompt"] != "go" {
		t.Fatalf("input was not preserved: %#v", body["input"])
	}

	parameters, ok := body["parameters"].(map[string]any)
	if !ok || parameters["size"] != "720P" {
		t.Fatalf("parameters were not preserved: %#v", body["parameters"])
	}
}

func TestAliNativeVideoHandlerPassesThroughAndStoresTask(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)

	store := &aliTestStore{}
	m := &meta.Meta{
		Mode:        mode.AliVideo,
		OriginModel: "wan2.5-t2v-preview",
		Group:       coremodel.GroupCache{ID: "group-1"},
		Token:       coremodel.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 42},
	}
	respBody := `{"request_id":"req-1","output":{"task_id":"task-123","task_status":"PENDING"}}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(respBody)),
	}

	result, relayErr := AliNativeVideoHandler(m, store, ctx, resp)
	if relayErr != nil {
		t.Fatalf("AliNativeVideoHandler returned error: %v", relayErr)
	}

	if recorder.Body.String() != respBody {
		t.Fatalf("unexpected passthrough body: %s", recorder.Body.String())
	}

	if result.UpstreamID != "task-123" || !result.AsyncUsage {
		t.Fatalf("unexpected result: %#v", result)
	}

	if len(store.saved) != 1 {
		t.Fatalf("expected one store save, got %d", len(store.saved))
	}

	if store.saved[0].ID != coremodel.VideoGenerationStoreID("task-123") ||
		store.saved[0].ChannelID != 42 ||
		store.saved[0].Model != "wan2.5-t2v-preview" {
		t.Fatalf("unexpected saved store: %#v", store.saved[0])
	}
}
