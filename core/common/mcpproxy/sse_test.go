package mcpproxy_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labring/aiproxy/core/common/mcpproxy"
)

type TestSessionManager struct {
	m map[string]string
}

func (t *TestSessionManager) New() string {
	return "test-session-id"
}

// Set stores a sessionID and its corresponding backend endpoint
func (t *TestSessionManager) Set(sessionID string, endpoint string) {
	t.m[sessionID] = endpoint
}

// Get retrieves the backend endpoint for a sessionID
func (t *TestSessionManager) Get(sessionID string) (string, bool) {
	v, ok := t.m[sessionID]
	return v, ok
}

// Delete removes a sessionID from the store
func (t *TestSessionManager) Delete(string) {
}

type TestEndpointHandler struct{}

func (h *TestEndpointHandler) NewEndpoint(_ string) string {
	return "/message?sessionId=test-session-id"
}

func (h *TestEndpointHandler) LoadEndpoint(endpoint string) string {
	if strings.Contains(endpoint, "test-session-id") {
		return "test-session-id"
	}
	return ""
}

func TestProxySSEEndpoint(t *testing.T) {
	reqDone := make(chan struct{})
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

		close(reqDone)
	}))
	defer backendServer.Close()

	// Create the proxy
	store := &TestSessionManager{
		m: map[string]string{},
	}
	handler := &TestEndpointHandler{}
	proxy := mcpproxy.NewSSEProxy(backendServer.URL+"/sse", nil, store, handler)

	// Setup the proxy server
	proxyServer := httptest.NewServer(http.HandlerFunc(proxy.SSEHandler))
	defer proxyServer.Close()

	// Make a request to the proxy
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, proxyServer.URL, nil)
	if err != nil {
		t.Fatalf("Error making request to proxy: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Error making request to proxy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	select {
	case <-time.NewTimer(time.Second).C:
		t.Error("timeout")
		return
	case <-reqDone:
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
