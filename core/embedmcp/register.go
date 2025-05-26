package embedmcp

import (
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/server"
)

var (
	servers            = make(map[string]EmbedMcp)
	mcpServerCache     = make(map[string]*server.MCPServer)
	mcpServerCacheLock = sync.RWMutex{}
)

func Register(mcp EmbedMcp) {
	if mcp.ID == "" {
		panic("mcp id is required")
	}
	if mcp.Name == "" {
		panic("mcp name is required")
	}
	if mcp.NewServer == nil {
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
		return getNoConfigServer(embedServer)
	}
	if err := ValidateConfigTemplatesConfig(embedServer.ConfigTemplates, config, reusingConfig); err != nil {
		return nil, fmt.Errorf("mcp %s config is invalid: %w", id, err)
	}
	return embedServer.NewServer(config, reusingConfig)
}

func getNoConfigServer(embedServer EmbedMcp) (*server.MCPServer, error) {
	mcpServerCacheLock.RLock()
	server, ok := mcpServerCache[embedServer.ID]
	mcpServerCacheLock.RUnlock()
	if ok {
		return server, nil
	}

	mcpServerCacheLock.Lock()
	defer mcpServerCacheLock.Unlock()
	server, ok = mcpServerCache[embedServer.ID]
	if ok {
		return server, nil
	}

	server, err := embedServer.NewServer(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp %s new server is invalid: %w", embedServer.ID, err)
	}
	mcpServerCache[embedServer.ID] = server
	return server, nil
}

func Servers() map[string]EmbedMcp {
	return servers
}

func GetEmbedMCP(id string) (EmbedMcp, bool) {
	mcp, ok := servers[id]
	return mcp, ok
}
