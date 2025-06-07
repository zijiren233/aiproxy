package fetch

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"fetch",
			"Fetch",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags([]string{"fetch", "web", "html", "markdown"}),
			mcpservers.WithReadme(readme),
		),
	)
}
