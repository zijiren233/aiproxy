package firecrawl

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"firecrawl",
			"Firecrawl",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/mendableai/firecrawl-mcp-server",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags(
				[]string{"firecrawl", "web", "scraping", "crawling", "search", "extraction"},
			),
			mcpservers.WithReadme(readme),
		),
	)
}
