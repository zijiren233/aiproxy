package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/mcpproxy"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/mark3labs/mcp-go/server"
)

type groupMcpEndpointProvider struct {
	key string
	t   model.GroupMCPType
}

func newGroupMcpEndpoint(key string, t model.GroupMCPType) mcpproxy.EndpointProvider {
	return &groupMcpEndpointProvider{
		key: key,
		t:   t,
	}
}

func (m *groupMcpEndpointProvider) NewEndpoint() (newSession string, newEndpoint string) {
	session := common.ShortUUID()
	endpoint := fmt.Sprintf("/mcp/group/message?sessionId=%s&key=%s&type=%s", session, m.key, m.t)
	return session, endpoint
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
//	@Router		/mcp/group/{id}/sse [get]
func GroupMCPSseServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "MCP ID, Group ID, and Session are required")
		return
	}

	group := middleware.GetGroup(c)

	mcp, err := model.GetGroupMCPByID(id, group.ID)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	switch mcp.Type {
	case model.GroupMCPTypeProxySSE:
		handleGroupProxySSE(c, mcp.ProxySSEConfig)
	case model.GroupMCPTypeOpenAPI:
		server, err := newOpenAPIMCPServer(mcp.OpenAPIConfig)
		if err != nil {
			middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
			return
		}
		handleGroupMCPServer(c, server, model.GroupMCPTypeOpenAPI)
	default:
		middleware.ErrorResponse(c, http.StatusBadRequest, "Unsupported MCP type")
	}
}

// handlePublicProxySSE processes SSE proxy requests
func handleGroupProxySSE(c *gin.Context, config *model.GroupMCPProxySSEConfig) {
	if config == nil || config.URL == "" {
		return
	}

	backendURL, err := url.Parse(config.URL)
	if err != nil {
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}

	headers := make(map[string]string)
	backendQuery := &url.Values{}
	token := middleware.GetToken(c)

	for k, v := range config.Headers {
		headers[k] = v
	}
	for k, v := range config.Querys {
		backendQuery.Set(k, v)
	}

	backendURL.RawQuery = backendQuery.Encode()
	mcpproxy.SSEHandler(
		c.Writer,
		c.Request,
		getStore(),
		newPublicMcpEndpoint(token.Key, model.PublicMCPTypeProxySSE),
		backendURL.String(),
		headers,
	)
}

// handleMCPServer handles the SSE connection for an MCP server
func handleGroupMCPServer(c *gin.Context, s *server.MCPServer, mcpType model.GroupMCPType) {
	token := middleware.GetToken(c)

	newSession, newEndpoint := newGroupMcpEndpoint(token.Key, mcpType).NewEndpoint()
	server := NewSSEServer(
		s,
		WithMessageEndpoint(newEndpoint),
	)

	// Store the session
	store := getStore()
	store.Set(newSession, string(mcpType))
	defer func() {
		store.Delete(newSession)
	}()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Start message processing goroutine
	go processMCPSseMpscMessages(ctx, newSession, server)

	// Handle SSE connection
	server.HandleSSE(c.Writer, c.Request)
}

// GroupMCPMessage godoc
//
//	@Summary	MCP SSE Proxy
//	@Router		/mcp/group/message [post]
func GroupMCPMessage(c *gin.Context) {
	token := middleware.GetToken(c)
	mcpTypeStr, _ := c.GetQuery("type")
	if mcpTypeStr == "" {
		return
	}
	mcpType := model.GroupMCPType(mcpTypeStr)
	sessionID, _ := c.GetQuery("sessionId")
	if sessionID == "" {
		return
	}

	switch mcpType {
	case model.GroupMCPTypeProxySSE:
		mcpproxy.ProxyHandler(
			c.Writer,
			c.Request,
			getStore(),
			newGroupMcpEndpoint(token.Key, mcpType),
		)
	default:
		sendMCPSSEMessage(c, mcpTypeStr, sessionID)
	}
}
