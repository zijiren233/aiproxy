package jinatools

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"jina",
			"Jina AI Tools",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/PsychArch/jina-mcp-tools",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags([]string{"jina", "web", "reader", "search", "fact-check", "ai"}),
			mcpservers.WithReadme(readme),
		),
	)
}
