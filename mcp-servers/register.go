package mcpservers

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

type mcpServerCacheItem struct {
	MCPServer         *server.MCPServer
	LastUsedTimestamp atomic.Int64
}

var (
	servers            = make(map[string]EmbedMcp)
	mcpServerCache     = make(map[string]*mcpServerCacheItem)
	mcpServerCacheLock = sync.RWMutex{}
)

func Register(mcp EmbedMcp) {
	if mcp.ID == "" {
		panic("mcp id is required")
	}
	if mcp.Name == "" {
		panic("mcp name is required")
	}
	if mcp.newServer == nil {
		panic(fmt.Sprintf("mcp %s new server is required", mcp.ID))
	}
	if mcp.ConfigTemplates != nil {
		if err := CheckConfigTemplatesValidate(mcp.ConfigTemplates); err != nil {
			panic(fmt.Sprintf("mcp %s config templates example is invalid: %v", mcp.ID, err))
		}
	}
	if _, ok := servers[mcp.ID]; ok {
		panic(fmt.Sprintf("mcp %s already registered", mcp.ID))
	}
	servers[mcp.ID] = mcp
}

func GetMCPServer(id string, config map[string]string, reusingConfig map[string]string) (*server.MCPServer, error) {
	embedServer, ok := servers[id]
	if !ok {
		return nil, fmt.Errorf("mcp %s not found", id)
	}
	if len(embedServer.ConfigTemplates) == 0 {
		return loadCacheServer(embedServer, nil)
	}

	if err := ValidateConfigTemplatesConfig(embedServer.ConfigTemplates, config, reusingConfig); err != nil {
		return nil, fmt.Errorf("mcp %s config is invalid: %w", id, err)
	}

	for _, template := range embedServer.ConfigTemplates {
		switch template.Required {
		case ConfigRequiredTypeReusingOptional, ConfigRequiredTypeReusingOnly, ConfigRequiredTypeInitOrReusingOnly:
			return embedServer.NewServer(config, reusingConfig)
		}
	}

	return loadCacheServer(embedServer, config)
}

func buildNoReusingConfigCacheKey(config map[string]string) string {
	keys := make([]string, 0, len(config))
	for key, value := range config {
		keys = append(keys, fmt.Sprintf("%s:%s", key, value))
	}
	sort.Strings(keys)
	return strings.Join(keys, ":")
}

func loadCacheServer(embedServer EmbedMcp, config map[string]string) (*server.MCPServer, error) {
	cacheKey := embedServer.ID
	if len(config) > 0 {
		cacheKey = fmt.Sprintf("%s:%s", embedServer.ID, buildNoReusingConfigCacheKey(config))
	}
	mcpServerCacheLock.RLock()
	server, ok := mcpServerCache[cacheKey]
	mcpServerCacheLock.RUnlock()
	if ok {
		server.LastUsedTimestamp.Store(time.Now().Unix())
		return server.MCPServer, nil
	}

	mcpServerCacheLock.Lock()
	defer mcpServerCacheLock.Unlock()
	server, ok = mcpServerCache[cacheKey]
	if ok {
		server.LastUsedTimestamp.Store(time.Now().Unix())
		return server.MCPServer, nil
	}

	mcpServer, err := embedServer.NewServer(config, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp %s new server is invalid: %w", embedServer.ID, err)
	}
	mcpServerCacheItem := &mcpServerCacheItem{
		MCPServer:         mcpServer,
		LastUsedTimestamp: atomic.Int64{},
	}
	mcpServerCacheItem.LastUsedTimestamp.Store(time.Now().Unix())
	mcpServerCache[cacheKey] = mcpServerCacheItem
	return mcpServer, nil
}

func Servers() map[string]EmbedMcp {
	return servers
}

func GetEmbedMCP(id string) (EmbedMcp, bool) {
	mcp, ok := servers[id]
	return mcp, ok
}
