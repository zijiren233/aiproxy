package mcpproxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EndpointProvider interface {
	NewEndpoint() (newSession string, newEndpoint string)
	LoadEndpoint(endpoint string) (session string)
}

// Proxy represents the proxy object that handles SSE and HTTP requests
type Proxy struct {
	store    SessionManager
	endpoint EndpointProvider
	backend  string
	headers  map[string]string
}

// NewProxy creates a new proxy with the given backend and endpoint handler
func NewProxy(backend string, headers map[string]string, store SessionManager, endpoint EndpointProvider) *Proxy {
	return &Proxy{
		store:    store,
		endpoint: endpoint,
		backend:  backend,
		headers:  headers,
	}
}

func (p *Proxy) SSEHandler(w http.ResponseWriter, r *http.Request) {
	SSEHandler(w, r, p.store, p.endpoint, p.backend, p.headers)
}

func SSEHandler(
	w http.ResponseWriter,
	r *http.Request,
	store SessionManager,
	endpoint EndpointProvider,
	backend string,
	headers map[string]string,
) {
	// Create a request to the backend SSE endpoint
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, backend, nil)
	if err != nil {
		http.Error(w, "Failed to create backend request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	// Set necessary headers for SSE
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Make the request to the backend
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to connect to backend", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Set SSE headers for the client response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a context that cancels when the client disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Monitor client disconnection
	go func() {
		<-ctx.Done()
		resp.Body.Close()
	}()

	// Parse the SSE stream and extract sessionId
	reader := bufio.NewReader(resp.Body)
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		// Write the line to the client
		fmt.Fprint(w, line)
		flusher.Flush()

		// Check if this is an endpoint event with sessionId
		if strings.HasPrefix(line, "event: endpoint") {
			// Next line should contain the data
			dataLine, err := reader.ReadString('\n')
			if err != nil {
				return
			}

			newSession, newEndpoint := endpoint.NewEndpoint()
			defer func() {
				store.Delete(newSession)
			}()

			// Extract sessionId from data line
			// Example: data: /message?sessionId=3088a771-7961-44e8-9bdf-21953889f694
			if strings.HasPrefix(dataLine, "data: ") {
				endpoint := strings.TrimSpace(strings.TrimPrefix(dataLine, "data: "))
				copyURL := *req.URL
				backendHostURL := &copyURL
				backendHostURL.Path = ""
				backendHostURL.RawQuery = ""
				store.Set(newSession, backendHostURL.String()+endpoint)
			} else {
				break
			}

			// Write the data line to the client
			fmt.Fprintf(w, "data: %s\n", newEndpoint)
			flusher.Flush()
		}
	}
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	ProxyHandler(w, r, p.store, p.endpoint)
}

func ProxyHandler(
	w http.ResponseWriter,
	r *http.Request,
	store SessionManager,
	endpoint EndpointProvider,
) {
	// Extract sessionID from the request
	sessionID := endpoint.LoadEndpoint(r.URL.String())
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// Look up the backend endpoint
	backendEndpoint, ok := store.Get(sessionID)
	if !ok {
		http.Error(w, "Invalid or expired sessionId", http.StatusNotFound)
		return
	}

	// Create a request to the backend
	req, err := http.NewRequestWithContext(r.Context(), r.Method, backendEndpoint, r.Body)
	if err != nil {
		http.Error(w, "Failed to create backend request", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	// Make the request to the backend
	client := &http.Client{
		Timeout: time.Second * 30,
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to connect to backend", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set response status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, _ = io.Copy(w, resp.Body)
}
