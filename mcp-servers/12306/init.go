// File: mcp-servers/12306/init.go
package train12306

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

func init() {
	mcpservers.Register(
		mcpservers.NewMcp(
			"12306",
			"12306 Train Ticket Query",
			mcpservers.McpTypeEmbed,
			mcpservers.WithNewServerFunc(NewServer),
			mcpservers.WithGitHubURL(
				"https://github.com/Joooook/12306-mcp",
			),
			mcpservers.WithConfigTemplates(configTemplates),
			mcpservers.WithTags([]string{"train", "12306", "ticket", "china", "railway"}),
			mcpservers.WithReadme(readme),
		),
	)
}
