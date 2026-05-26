//nolint:testpackage
package doubao

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

func TestConvertNativeVideoRequestPreservesBodyAndRewritesModel(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v3/contents/generations/tasks",
		bytes.NewBufferString(
			`{"model":"doubao-seedance-2-0","content":[{"type":"text","text":"go"}],"resolution":"720p"}`,
		),
	)
	req.Header.Set("Content-Type", "application/json")

	result, err := ConvertDoubaoNativeVideoRequest(&meta.Meta{
		ActualModel: "mapped-seedance",
	}, req)
	if err != nil {
		t.Fatalf("ConvertDoubaoNativeVideoRequest returned error: %v", err)
	}

	var body map[string]any
	if err := json.NewDecoder(result.Body).Decode(&body); err != nil {
		t.Fatalf("decode converted body: %v", err)
	}

	if body["model"] != "mapped-seedance" {
		t.Fatalf("model was not rewritten: %#v", body["model"])
	}

	if body["resolution"] != "720p" {
		t.Fatalf("resolution was not preserved: %#v", body["resolution"])
	}

	content, ok := body["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content was not preserved: %#v", body["content"])
	}
}

func TestDoubaoNativeVideoSubmitHandlerPassesThroughAndStoresTask(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)

	store := &doubaoTestStore{}
	m := &meta.Meta{
		Mode:        mode.DoubaoVideo,
		OriginModel: "doubao-seedance-2-0",
		Group:       coremodel.GroupCache{ID: "group-1"},
		Token:       coremodel.TokenCache{ID: 7},
		Channel:     meta.ChannelMeta{ID: 42},
	}
	respBody := `{"id":"task-123","model":"doubao-seedance-2-0","status":"queued"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(respBody)),
	}

	result, relayErr := DoubaoNativeVideoSubmitHandler(m, store, ctx, resp)
	if relayErr != nil {
		t.Fatalf("DoubaoNativeVideoSubmitHandler returned error: %v", relayErr)
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
		store.saved[0].Model != "doubao-seedance-2-0" {
		t.Fatalf("unexpected saved store: %#v", store.saved[0])
	}
}
