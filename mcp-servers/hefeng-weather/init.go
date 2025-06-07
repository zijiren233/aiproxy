package hefengweather

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"hefeng-weather",
			"HeFeng Weather",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/shanggqm/hefeng-mcp-weather",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags([]string{"weather", "天气", "和风天气", "forecast", "china"}),
			mcpservers.WithReadme(readme),
		),
	)
}
