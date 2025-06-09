package gezhe

import (
	_ "embed"

	"github.com/labring/aiproxy/core/model"
	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

//go:embed README.cn.md
var readmeCN string

// need import in mcpregister/init.go
func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"gezhe",
			"Gezhe",
			model.PublicMCPTypeProxyStreamable,
			mcpservers.WithProxyConfigType(configTemplates),
			mcpservers.WithTags([]string{"map"}),
			mcpservers.WithDescription(
				"可以通过话题生成 PPT",
			),
			mcpservers.WithReadme(readme),
			mcpservers.WithReadmeCN(readmeCN),
		),
	)
}
