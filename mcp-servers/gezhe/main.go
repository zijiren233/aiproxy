package gezhe

import (
	"github.com/labring/aiproxy/core/model"
	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

const defaultURL = "https://mcp.gezhe.com/mcp"

var configTemplates = map[string]mcpservers.ConfigTemplate{
	"API_KEY": {
		Name:        "API_KEY",
		Required:    mcpservers.ConfigRequiredTypeInitOrReusingOnly,
		Example:     "bx7Qt1BLbxRq...",
		Description: "The key of the Gezhe MCP server: https://pro.gezhe.com/settings",
	},

	"url": {
		Name:        "URL",
		Required:    mcpservers.ConfigRequiredTypeInitOptional,
		Example:     defaultURL,
		Description: "The Streamable http URL of the gezhe MCP server",
	},
}

var proxyConfigType = map[string]model.ProxyParamType{
	"API_KEY": model.ParamTypeQuery,
	"url":     model.ParamTypeURL,
}
