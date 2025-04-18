package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

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
	server := &http.Server{Addr: ":3000", ReadHeaderTimeout: time.Second * 10}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
