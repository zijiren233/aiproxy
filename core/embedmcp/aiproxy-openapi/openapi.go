package aiproxyopenapi

import (
	"fmt"
	"net/url"

	"github.com/labring/aiproxy/core/docs"
	"github.com/labring/aiproxy/core/embedmcp"
	"github.com/labring/aiproxy/openapi-mcp/convert"
	"github.com/mark3labs/mcp-go/server"
)

var configTemplates = map[string]embedmcp.ConfigTemplate{
	"host": {
		Name:        "host",
		Required:    embedmcp.ConfigRequiredTypeInitOnly,
		Example:     "http://localhost:3000",
		Description: "The host of the OpenAPI server",
		Validator: func(value string) error {
			u, err := url.Parse(value)
			if err != nil {
				return err
			}
			if u.Scheme != "http" && u.Scheme != "https" {
				return fmt.Errorf("invalid scheme: %s", u.Scheme)
			}
			return nil
		},
	},
}

func NewServer(config map[string]string, _ map[string]string) (*server.MCPServer, error) {
	parser := convert.NewParser()
	err := parser.Parse([]byte(docs.SwaggerInfo.ReadDoc()))
	if err != nil {
		return nil, err
	}
	converter := convert.NewConverter(parser, convert.Options{
		OpenAPIFrom: config["host"],
	})
	return converter.Convert()
}

// need import in mcpregister/init.go
func init() {
	embedmcp.Register(embedmcp.EmbedMcp{
		ID:              "aiproxy-openapi",
		Name:            "AI Proxy OpenAPI",
		NewServer:       NewServer,
		ConfigTemplates: configTemplates,
	})
}
