package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labring/aiproxy/core/common/mcpproxy"
)

type TestEndpointHandler struct{}

func (h *TestEndpointHandler) NewEndpoint() (string, string) {
	return "test-session-id", "/test?sessionId=test-session-id"
}

func (h *TestEndpointHandler) LoadEndpoint(endpoint string) string {
	if strings.Contains(endpoint, "test-session-id") {
		return "test-session-id"
	}
	return ""
}

func main() {
	// Start the proxy server on port 3000
	store := mcpproxy.NewMemStore()
	handler := &TestEndpointHandler{}
	proxy := mcpproxy.NewProxy("http://localhost:3001/sse", nil, store, handler)

	// Setup routes
	http.HandleFunc("/sse", proxy.SSEHandler)
	http.HandleFunc("/test", proxy.ProxyHandler)

	// Start the server in a goroutine
	fmt.Println("Starting proxy server on :3000")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
