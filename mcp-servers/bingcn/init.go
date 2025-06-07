package bingcn

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"bing-cn-search",
			"Bing CN Search",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/yan5236/bing-cn-mcp-server",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags([]string{"search", "bing", "web", "scraping"}),
			mcpservers.WithReadme(readme),
		),
	)
}
