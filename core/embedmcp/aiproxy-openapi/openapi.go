package aiproxyopenapi

import (
	"github.com/labring/aiproxy/core/docs"
	"github.com/labring/aiproxy/core/embedmcp"
	"github.com/labring/aiproxy/openapi-mcp/convert"
	"github.com/mark3labs/mcp-go/server"
)

func NewServer(config map[string]string, reusingConfig map[string]string) (*server.MCPServer, error) {
	parser := convert.NewParser()
	err := parser.Parse([]byte(docs.SwaggerInfo.ReadDoc()))
	if err != nil {
		return nil, err
	}
	converter := convert.NewConverter(parser, convert.Options{})
	return converter.Convert()
}

// need import in mcpregister/init.go
func init() {
	embedmcp.Register(embedmcp.EmbedMcp{
		ID:        "aiproxy-openapi",
		Name:      "AI Proxy OpenAPI",
		NewServer: NewServer,
	})
}
