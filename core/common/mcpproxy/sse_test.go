package mcpproxy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labring/aiproxy/core/common/mcpproxy"
)

type TestEndpointHandler struct{}

func (h *TestEndpointHandler) NewEndpoint() (string, string) {
	return "test-session-id", "/message?sessionId=test-session-id"
}

func (h *TestEndpointHandler) LoadEndpoint(endpoint string) string {
	if strings.Contains(endpoint, "test-session-id") {
		return "test-session-id"
	}
	return ""
}

func TestProxySSEEndpoint(t *testing.T) {
	// Setup a mock backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected ResponseWriter to be a Flusher")
		}

		// Send an endpoint event
		fmt.Fprintf(w, "event: endpoint\n")
		fmt.Fprintf(w, "data: /message?sessionId=original-session-id\n\n")
		flusher.Flush()

		// Keep the connection open for a bit
		time.Sleep(100 * time.Millisecond)
	}))
	defer backendServer.Close()

	// Create the proxy
	store := mcpproxy.NewMemStore()
	handler := &TestEndpointHandler{}
	proxy := mcpproxy.NewProxy(backendServer.URL+"/sse", nil, store, handler)

	// Setup the proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(proxy.SSEHandler))
	defer proxyServer.Close()

	// Make a request to the proxy
	resp, err := http.Get(proxyServer.URL)
	if err != nil {
		t.Fatalf("Error making request to proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify the session was stored
	endpoint, ok := store.Get("test-session-id")
	if !ok {
		t.Error("Session was not stored")
	}
	if !strings.Contains(endpoint, "/message?sessionId=original-session-id") {
		t.Errorf("Endpoint does not contain expected path, got: %s", endpoint)
	}
}
