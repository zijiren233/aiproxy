package aiproxyopenapi

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/labring/aiproxy/core/docs"
	"github.com/labring/aiproxy/core/embedmcp"
	"github.com/labring/aiproxy/openapi-mcp/convert"
	"github.com/mark3labs/mcp-go/server"
)

var configTemplates = map[string]embedmcp.ConfigTemplate{
	"host": {
		Name:        "Host",
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

	"authorization": {
		Name:        "Authorization",
		Required:    embedmcp.ConfigRequiredTypeReusingOptional,
		Example:     "aiproxy-admin-key",
		Description: "The admin key of the OpenAPI server",
	},
}

var (
	parser    *convert.Parser
	parseOnce sync.Once
)

func getParser() *convert.Parser {
	parseOnce.Do(func() {
		parser = convert.NewParser()
		err := parser.Parse([]byte(docs.SwaggerInfo.ReadDoc()))
		if err != nil {
			panic(err)
		}
	})
	return parser
}

func NewServer(config map[string]string, reusingConfig map[string]string) (*server.MCPServer, error) {
	converter := convert.NewConverter(getParser(), convert.Options{
		OpenAPIFrom:   config["host"],
		Authorization: reusingConfig["authorization"],
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
