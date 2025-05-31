package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common/mcpproxy"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type groupMcpEndpointProvider struct {
	key string
	t   model.GroupMCPType
}

func newGroupMcpEndpoint(key string, t model.GroupMCPType) EndpointProvider {
	return &groupMcpEndpointProvider{
		key: key,
		t:   t,
	}
}

func (m *groupMcpEndpointProvider) NewEndpoint(session string) (newEndpoint string) {
	endpoint := fmt.Sprintf("/mcp/group/message?sessionId=%s&key=%s&type=%s", session, m.key, m.t)
	return endpoint
}

func (m *groupMcpEndpointProvider) LoadEndpoint(endpoint string) (session string) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	return parsedURL.Query().Get("sessionId")
}

// GroupMCPSseServer godoc
//
//	@Summary	Group MCP SSE Server
//	@Security	ApiKeyAuth
//	@Router		/mcp/group/{id}/sse [get]
func GroupMCPSseServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.Error(c.Writer, "mcp id is required", http.StatusBadRequest)
		return
	}

	group := middleware.GetGroup(c)

	groupMcp, err := model.CacheGetGroupMCP(group.ID, id)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusNotFound)
		return
	}
	if groupMcp.Status != model.GroupMCPStatusEnabled {
		http.Error(c.Writer, "mcp is not enabled", http.StatusNotFound)
		return
	}

	switch groupMcp.Type {
	case model.GroupMCPTypeProxySSE:
		client, err := transport.NewSSE(
			groupMcp.ProxyConfig.URL,
			transport.WithHeaders(groupMcp.ProxyConfig.Headers),
		)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}
		err = client.Start(c.Request.Context())
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}
		defer client.Close()
		handleGroupMCPServer(c,
			mcpproxy.WrapMCPClient2Server(client),
			model.GroupMCPTypeProxySSE,
		)
	case model.GroupMCPTypeProxyStreamable:
		client, err := transport.NewStreamableHTTP(
			groupMcp.ProxyConfig.URL,
			transport.WithHTTPHeaders(groupMcp.ProxyConfig.Headers),
		)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}
		err = client.Start(c.Request.Context())
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}
		defer client.Close()
		handleGroupMCPServer(
			c,
			mcpproxy.WrapMCPClient2Server(client),
			model.GroupMCPTypeProxyStreamable,
		)
	case model.GroupMCPTypeOpenAPI:
		server, err := newOpenAPIMCPServer(groupMcp.OpenAPIConfig)
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusBadRequest)
			return
		}
		handleGroupMCPServer(c, server, model.GroupMCPTypeOpenAPI)
	default:
		http.Error(c.Writer, "unsupported mcp type", http.StatusBadRequest)
	}
}

// handleMCPServer handles the SSE connection for an MCP server
func handleGroupMCPServer(c *gin.Context, s mcpproxy.MCPServer, mcpType model.GroupMCPType) {
	token := middleware.GetToken(c)

	// Store the session
	store := getStore()
	newSession := store.New()

	newEndpoint := newGroupMcpEndpoint(token.Key, mcpType).NewEndpoint(newSession)
	server := mcpproxy.NewSSEServer(
		s,
		mcpproxy.WithMessageEndpoint(newEndpoint),
	)

	store.Set(newSession, string(mcpType))
	defer func() {
		store.Delete(newSession)
	}()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Start message processing goroutine
	go processMCPSseMpscMessages(ctx, newSession, server)

	// Handle SSE connection
	server.ServeHTTP(c.Writer, c.Request)
}

// GroupMCPMessage godoc
//
//	@Summary	MCP SSE Proxy
//	@Security	ApiKeyAuth
//	@Router		/mcp/group/message [post]
func GroupMCPMessage(c *gin.Context) {
	mcpTypeStr, _ := c.GetQuery("type")
	if mcpTypeStr == "" {
		http.Error(c.Writer, "missing mcp type", http.StatusBadRequest)
		return
	}
	mcpType := model.GroupMCPType(mcpTypeStr)

	sessionID, _ := c.GetQuery("sessionId")
	if sessionID == "" {
		http.Error(c.Writer, "missing sessionId", http.StatusBadRequest)
		return
	}

	switch mcpType {
	case model.GroupMCPTypeProxySSE:
		sendMCPSSEMessage(c, mcpTypeStr, sessionID)
	case model.GroupMCPTypeProxyStreamable:
		sendMCPSSEMessage(c, mcpTypeStr, sessionID)
	case model.GroupMCPTypeOpenAPI:
		sendMCPSSEMessage(c, mcpTypeStr, sessionID)
	default:
		http.Error(c.Writer, "unknown mcp type", http.StatusBadRequest)
	}
}

// GroupMCPStreamable godoc
//
//	@Summary	Group MCP Streamable Server
//	@Security	ApiKeyAuth
//	@Router		/mcp/group/{id}/streamable [get]
//	@Router		/mcp/group/{id}/streamable [post]
//	@Router		/mcp/group/{id}/streamable [delete]
func GroupMCPStreamable(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, mcpproxy.CreateMCPErrorResponse(
			mcp.NewRequestId(nil),
			mcp.INVALID_REQUEST,
			"mcp id is required",
		))
		return
	}

	group := middleware.GetGroup(c)

	groupMcp, err := model.CacheGetGroupMCP(group.ID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, mcpproxy.CreateMCPErrorResponse(
			mcp.NewRequestId(nil),
			mcp.INVALID_REQUEST,
			err.Error(),
		))
		return
	}
	if groupMcp.Status != model.GroupMCPStatusEnabled {
		c.JSON(http.StatusNotFound, mcpproxy.CreateMCPErrorResponse(
			mcp.NewRequestId(nil),
			mcp.INVALID_REQUEST,
			"mcp is not enabled",
		))
		return
	}

	switch groupMcp.Type {
	case model.GroupMCPTypeProxyStreamable:
		handleGroupProxyStreamable(c, groupMcp.ProxyConfig)
	case model.GroupMCPTypeOpenAPI:
		server, err := newOpenAPIMCPServer(groupMcp.OpenAPIConfig)
		if err != nil {
			c.JSON(http.StatusBadRequest, mcpproxy.CreateMCPErrorResponse(
				mcp.NewRequestId(nil),
				mcp.INVALID_REQUEST,
				err.Error(),
			))
			return
		}
		handleStreamableMCPServer(c, server)
	default:
		c.JSON(http.StatusBadRequest, mcpproxy.CreateMCPErrorResponse(
			mcp.NewRequestId(nil),
			mcp.INVALID_REQUEST,
			"unsupported mcp type",
		))
	}
}

// handleGroupProxyStreamable processes Streamable proxy requests for group
func handleGroupProxyStreamable(c *gin.Context, config *model.GroupMCPProxyConfig) {
	if config == nil || config.URL == "" {
		return
	}

	backendURL, err := url.Parse(config.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, mcpproxy.CreateMCPErrorResponse(
			mcp.NewRequestId(nil),
			mcp.INVALID_REQUEST,
			err.Error(),
		))
		return
	}

	headers := make(map[string]string)
	backendQuery := backendURL.Query()

	for k, v := range config.Headers {
		headers[k] = v
	}
	for k, v := range config.Querys {
		backendQuery.Set(k, v)
	}

	backendURL.RawQuery = backendQuery.Encode()
	mcpproxy.NewStreamableProxy(backendURL.String(), headers, getStore()).
		ServeHTTP(c.Writer, c.Request)
}
