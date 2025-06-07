package baidumap

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"baidu-map",
			"Baidu Map",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/baidu-maps/mcp",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags(
				[]string{"baidu", "map", "geocoding", "search", "weather", "traffic"},
			),
			mcpservers.WithReadme(readme),
		),
	)
}
