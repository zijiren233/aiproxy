package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/mcpproxy"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/openapi-mcp/convert"
	"github.com/redis/go-redis/v9"
)

// mcpEndpointProvider implements the EndpointProvider interface for MCP
type mcpEndpointProvider struct {
	key string
	t   model.PublicMCPType
}

func newEndpoint(key string, t model.PublicMCPType) mcpproxy.EndpointProvider {
	return &mcpEndpointProvider{
		key: key,
		t:   t,
	}
}

func (m *mcpEndpointProvider) NewEndpoint() (newSession string, newEndpoint string) {
	session := uuid.NewString()
	endpoint := fmt.Sprintf("/mcp/message?sessionId=%s&key=%s&type=%s", session, m.key, m.t)
	return session, endpoint
}

func (m *mcpEndpointProvider) LoadEndpoint(endpoint string) (session string) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	return parsedURL.Query().Get("sessionId")
}

// Global variables for session management
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

// Redis-based session manager
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

func (r *redisStoreManager) Get(sessionID string) (string, bool) {
	ctx := context.Background()

	result, err := redisStoreManagerScript.Run(ctx, r.rdb, []string{"mcp:session:" + sessionID}).Result()
	if err != nil || result == nil {
		return "", false
	}

	return result.(string), true
}

func (r *redisStoreManager) Set(sessionID, endpoint string) {
	ctx := context.Background()
	r.rdb.Set(ctx, "mcp:session:"+sessionID, endpoint, time.Minute*5)
}

func (r *redisStoreManager) Delete(session string) {
	ctx := context.Background()
	r.rdb.Del(ctx, "mcp:session:"+session)
}

// MCPSseProxy godoc
//
//	@Summary	MCP SSE Proxy
//	@Router		/mcp/public/{id}/sse [get]
func MCPSseProxy(c *gin.Context) {
	mcpId := c.Param("id")

	publicMcp, err := model.GetPublicMCPByID(mcpId)
	if err != nil {
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}

	switch publicMcp.Type {
	case model.PublicMCPTypeProxySSE:
		handleProxySSE(c, publicMcp)
	case model.PublicMCPTypeOpenAPI:
		handleOpenAPI(c, publicMcp)
	default:
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, "unknow mcp type")
		return
	}
}

// handleProxySSE processes SSE proxy requests
func handleProxySSE(c *gin.Context, publicMcp *model.PublicMCP) {
	config := publicMcp.ProxySSEConfig
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
	group := middleware.GetGroup(c)
	token := middleware.GetToken(c)

	// Process reusing parameters if any
	if err := processReusingParams(config.ReusingParams, publicMcp.ID, group.ID, headers, backendQuery); err != nil {
		middleware.AbortLogWithMessage(c, http.StatusBadRequest, err.Error())
		return
	}

	backendURL.RawQuery = backendQuery.Encode()
	mcpproxy.SSEHandler(
		c.Writer,
		c.Request,
		getStore(),
		newEndpoint(token.Key, publicMcp.Type),
		backendURL.String(),
		headers,
	)
}

// handleOpenAPI processes OpenAPI requests
func handleOpenAPI(c *gin.Context, publicMcp *model.PublicMCP) {
	config := publicMcp.OpenAPIConfig
	if config == nil || (config.OpenAPISpec == "" && config.OpenAPIContent == "") {
		return
	}

	// Parse OpenAPI specification
	parser := convert.NewParser()
	var err error
	var openAPIFrom string

	if config.OpenAPISpec != "" {
		openAPIFrom, err = parseOpenAPIFromURL(config, parser)
	} else {
		err = parseOpenAPIFromContent(config, parser)
	}

	if err != nil {
		return
	}

	// Convert to MCP server
	converter := convert.NewConverter(parser, convert.Options{
		OpenAPIFrom: openAPIFrom,
	})
	s, err := converter.Convert()
	if err != nil {
		return
	}

	token := middleware.GetToken(c)

	// Setup SSE server
	newSession, newEndpoint := newEndpoint(token.Key, publicMcp.Type).NewEndpoint()
	store := getStore()
	store.Set(newSession, "openapi")
	defer func() {
		store.Delete(newSession)
	}()

	server := NewSSEServer(
		s,
		WithMessageEndpoint(newEndpoint),
	)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Start message processing goroutine
	go processOpenAPIMessages(ctx, newSession, server)

	// Handle SSE connection
	server.HandleSSE(c.Writer, c.Request)
}

// parseOpenAPIFromURL parses OpenAPI spec from a URL
func parseOpenAPIFromURL(config *model.MCPOpenAPIConfig, parser *convert.Parser) (string, error) {
	spec, err := url.Parse(config.OpenAPISpec)
	if err != nil || (spec.Scheme != "http" && spec.Scheme != "https") {
		return "", errors.New("invalid OpenAPI spec URL")
	}

	openAPIFrom := spec.String()
	if config.V2 {
		err = parser.ParseFileV2(openAPIFrom)
	} else {
		err = parser.ParseFile(openAPIFrom)
	}

	return openAPIFrom, err
}

// parseOpenAPIFromContent parses OpenAPI spec from content string
func parseOpenAPIFromContent(config *model.MCPOpenAPIConfig, parser *convert.Parser) error {
	if config.V2 {
		return parser.ParseV2([]byte(config.OpenAPIContent))
	}
	return parser.Parse([]byte(config.OpenAPIContent))
}

// processOpenAPIMessages handles message processing for OpenAPI
func processOpenAPIMessages(ctx context.Context, sessionID string, server *SSEServer) {
	mpscInstance := getMpsc()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := mpscInstance.recv(ctx, sessionID)
			if err != nil {
				return
			}
			if err := server.HandleMessage(data); err != nil {
				return
			}
		}
	}
}

// processReusingParams handles the reusing parameters for MCP proxy
func processReusingParams(reusingParams map[string]model.ReusingParam, mcpId string, groupID string, headers map[string]string, backendQuery *url.Values) error {
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
		default:
			return errors.New("unknow param type")
		}
	}

	return nil
}

// MCPMessage godoc
//
//	@Summary	MCP SSE Proxy
//	@Router		/mcp/message [post]
func MCPMessage(c *gin.Context) {
	token := middleware.GetToken(c)
	mcpTypeStr, _ := c.GetQuery("type")
	if mcpTypeStr == "" {
		return
	}
	mcpType := model.PublicMCPType(mcpTypeStr)
	sessionID, _ := c.GetQuery("sessionId")
	if sessionID == "" {
		return
	}

	switch mcpType {
	case model.PublicMCPTypeProxySSE:
		mcpproxy.ProxyHandler(
			c.Writer,
			c.Request,
			getStore(),
			newEndpoint(token.Key, mcpType),
		)
	case model.PublicMCPTypeOpenAPI:
		backend, ok := getStore().Get(sessionID)
		if !ok || backend != "openapi" {
			return
		}
		mpscInstance := getMpsc()
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		err = mpscInstance.send(c.Request.Context(), sessionID, body)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Writer.WriteHeader(http.StatusAccepted)
	}
}

// Interface for multi-producer, single-consumer message passing
type mpsc interface {
	recv(ctx context.Context, id string) ([]byte, error)
	send(ctx context.Context, id string, data []byte) error
}

// Global MPSC instances
var (
	memMpsc       mpsc = newChannelMpsc()
	redisMpsc     mpsc
	redisMpscOnce = &sync.Once{}
)

func getMpsc() mpsc {
	if common.RedisEnabled {
		redisMpscOnce.Do(func() {
			redisMpsc = NewRedisMPSC(common.RDB)
		})
		return redisMpsc
	}
	return memMpsc
}

// In-memory channel-based MPSC implementation
type channelMpsc struct {
	channels     map[string]chan []byte
	lastAccess   map[string]time.Time
	channelMutex sync.RWMutex
}

// newChannelMpsc creates a new channel-based mpsc implementation
func newChannelMpsc() *channelMpsc {
	c := &channelMpsc{
		channels:   make(map[string]chan []byte),
		lastAccess: make(map[string]time.Time),
	}

	// Start a goroutine to clean up expired channels
	go c.cleanupExpiredChannels()

	return c
}

// cleanupExpiredChannels periodically checks for and removes channels that haven't been accessed in 5 minutes
func (c *channelMpsc) cleanupExpiredChannels() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.channelMutex.Lock()
		now := time.Now()
		for id, lastAccess := range c.lastAccess {
			if now.Sub(lastAccess) > 5*time.Minute {
				// Close and delete the channel
				if ch, exists := c.channels[id]; exists {
					close(ch)
					delete(c.channels, id)
				}
				delete(c.lastAccess, id)
			}
		}
		c.channelMutex.Unlock()
	}
}

// getOrCreateChannel gets an existing channel or creates a new one for the session
func (c *channelMpsc) getOrCreateChannel(id string) chan []byte {
	c.channelMutex.RLock()
	ch, exists := c.channels[id]
	c.channelMutex.RUnlock()

	if !exists {
		c.channelMutex.Lock()
		if ch, exists = c.channels[id]; !exists {
			ch = make(chan []byte, 10)
			c.channels[id] = ch
		}
		c.lastAccess[id] = time.Now()
		c.channelMutex.Unlock()
	} else {
		c.channelMutex.Lock()
		c.lastAccess[id] = time.Now()
		c.channelMutex.Unlock()
	}

	return ch
}

// recv receives data for the specified session
func (c *channelMpsc) recv(ctx context.Context, id string) ([]byte, error) {
	ch := c.getOrCreateChannel(id)

	select {
	case data, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("channel closed for session %s", id)
		}
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// send sends data to the specified session
func (c *channelMpsc) send(ctx context.Context, id string, data []byte) error {
	ch := c.getOrCreateChannel(id)

	select {
	case ch <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("channel buffer full for session %s", id)
	}
}

// Redis-based MPSC implementation
type redisMPSC struct {
	rdb *redis.Client
}

// NewRedisMPSC creates a new Redis MPSC instance
func NewRedisMPSC(rdb *redis.Client) *redisMPSC {
	return &redisMPSC{rdb: rdb}
}

func (r *redisMPSC) send(ctx context.Context, id string, data []byte) error {
	// Set expiration to 5 minutes when sending data
	pipe := r.rdb.Pipeline()
	pipe.LPush(ctx, id, data)
	pipe.Expire(ctx, id, 5*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisMPSC) recv(ctx context.Context, id string) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			result, err := r.rdb.BRPop(ctx, time.Second, id).Result()
			if err != nil {
				if err == redis.Nil {
					runtime.Gosched()
					continue
				}
				return nil, err
			}
			if len(result) != 2 {
				return nil, errors.New("invalid BRPop result")
			}
			return []byte(result[1]), nil
		}
	}
}
