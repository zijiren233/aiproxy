package flomo

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"flomo",
			"Flomo",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/chatmcp/mcp-server-flomo",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags([]string{"flomo", "notes", "writing", "productivity"}),
			mcpservers.WithReadme(readme),
		),
	)
}
