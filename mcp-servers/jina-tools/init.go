package jinatools

import (
	_ "embed"

	mcpservers "github.com/labring/aiproxy/mcp-servers"
)

//go:embed README.md
var readme string

//go:embed README.cn.md
var readmeCN string

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
			mcpservers.WithDescription(
				"A Model Context Protocol (MCP) server that integrates with Jina AI Search Foundation APIs. Provides web reading, web search, and fact-checking capabilities.",
			),
			mcpservers.WithDescriptionCN("集成Jina AI搜索基础API的模型上下文协议(MCP)服务器。提供网页阅读、网络搜索和事实核查功能。"),
			mcpservers.WithReadmeCN(readmeCN),
			mcpservers.WithReadme(readme),
		),
	)
}
