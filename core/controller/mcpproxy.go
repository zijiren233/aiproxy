package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/mcpproxy"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/redis/go-redis/v9"
)

// mcpEndpointProvider implements the EndpointProvider interface for MCP
type mcpEndpointProvider struct {
	key string
}

func newEndpoint(key string) mcpproxy.EndpointProvider {
	return &mcpEndpointProvider{
		key: key,
	}
}

func (m *mcpEndpointProvider) NewEndpoint() (newSession string, newEndpoint string) {
	session := uuid.NewString()
	endpoint := fmt.Sprintf("/mcp/message?sessionId=%s&key=%s", session, m.key)
	return session, endpoint
}

func (m *mcpEndpointProvider) LoadEndpoint(endpoint string) (session string) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	return parsedURL.Query().Get("sessionId")
}

type redisStoreManager struct {
	rdb *redis.Client
}

func newRedisStoreManager(rdb *redis.Client) mcpproxy.SessionManager {
	return &redisStoreManager{
		rdb: rdb,
	}
}

var redisStoreManagerScript = redis.NewScript(`
local key = KEYS[1]
local value = redis.call('GET', key)
if not value then
	return nil
end
redis.call('EXPIRE', key, 300)
return value
`)

func (r *redisStoreManager) Get(sessionId string) (string, bool) {
	ctx := context.Background()

	result, err := redisStoreManagerScript.Run(ctx, r.rdb, []string{"mcp:session:" + sessionId}).Result()
	if err != nil || result == nil {
		return "", false
	}

	return result.(string), true
}

func (r *redisStoreManager) Set(sessionId, endpoint string) {
	ctx := context.Background()
	r.rdb.Set(ctx, "mcp:session:"+sessionId, endpoint, time.Minute*5)
}

func (r *redisStoreManager) Delete(session string) {
	ctx := context.Background()
	r.rdb.Del(ctx, "mcp:session:"+session)
}

// Global variables for MCP proxy
var (
	memStore       mcpproxy.SessionManager = mcpproxy.NewMemStore()
	redisStore     mcpproxy.SessionManager
	redisStoreOnce = &sync.Once{}
)

func getStore() mcpproxy.SessionManager {
	if common.RedisEnabled {
		redisStoreOnce.Do(func() {
			redisStore = newRedisStoreManager(common.RDB)
		})
		return redisStore
	}
	return memStore
}

// MCPSseProxy godoc
//
//	@Summary	MCP SSE Proxy
//	@Router		/mcp/public/{id}/sse [get]
func MCPSseProxy(c *gin.Context) {
	group := middleware.GetGroup(c)
	token := middleware.GetToken(c)
	mcpId := c.Param("id")

	publicMcp, err := model.GetPublicMCPByID(mcpId)
	if err != nil {
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}

	if publicMcp.Type != model.MCPTypeProxySSE {
		return
	}

	if publicMcp.ProxySSEConfig == nil || publicMcp.ProxySSEConfig.URL == "" {
		return
	}

	backendURL, err := url.Parse(publicMcp.ProxySSEConfig.URL)
	if err != nil {
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}

	headers := make(map[string]string)
	backendQuery := &url.Values{}

	// Process reusing parameters if any
	if err := processReusingParams(publicMcp, mcpId, group.ID, headers, backendQuery); err != nil {
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}

	backendURL.RawQuery = backendQuery.Encode()
	mcpproxy.SSEHandler(
		c.Writer,
		c.Request,
		getStore(),
		newEndpoint(token.Key),
		backendURL.String(),
		headers,
	)
}

// processReusingParams handles the reusing parameters for MCP proxy
func processReusingParams(publicMcp *model.PublicMCP, mcpId string, groupID string, headers map[string]string, backendQuery *url.Values) error {
	reusingParams := publicMcp.ProxySSEConfig.ReusingParams
	if len(reusingParams) == 0 {
		return nil
	}

	param, err := model.GetGroupPublicMCPReusingParam(mcpId, groupID)
	if err != nil {
		return err
	}

	for k, v := range reusingParams {
		paramValue, ok := param.ReusingParams[k]
		if !ok {
			if v.Required {
				return fmt.Errorf("%s required", k)
			}
			continue
		}

		switch v.Type {
		case model.ParamTypeHeader:
			headers[k] = paramValue
		case model.ParamTypeQuery:
			backendQuery.Set(k, paramValue)
		}
	}

	return nil
}

// MCPProxy godoc
//
//	@Summary	MCP SSE Proxy
//	@Router		/mcp/message [post]
func MCPProxy(c *gin.Context) {
	token := middleware.GetToken(c)
	mcpproxy.ProxyHandler(
		c.Writer,
		c.Request,
		getStore(),
		newEndpoint(token.Key),
	)
}
