package websearch

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptors"
	"github.com/labring/aiproxy/core/relay/controller"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	"github.com/labring/aiproxy/core/relay/plugin"
	"github.com/labring/aiproxy/core/relay/plugin/noop"
	"github.com/labring/aiproxy/core/relay/utils"
	"github.com/labring/aiproxy/mcp-servers/web-search/engine"
)

var _ plugin.Plugin = (*WebSearch)(nil)

type GetChannel func(modelName string) (*model.Channel, error)

// WebSearch implements web search functionality
type WebSearch struct {
	noop.Noop
	GetChannel GetChannel
}

// NewWebSearchPlugin creates a new web search plugin
func NewWebSearchPlugin(getChannel GetChannel) plugin.Plugin {
	return &WebSearch{
		GetChannel: getChannel,
	}
}

//go:embed prompts/arxiv.md
var arxivSearchPrompts string

//go:embed prompts/internet.md
var internetSearchPrompts string

// ConvertRequest intercepts and modifies requests to add web search capabilities
func (p *WebSearch) ConvertRequest(meta *meta.Meta, req *http.Request, do adaptor.ConvertRequest) (*adaptor.ConvertRequestResult, error) {
	// Skip if not chat completions mode
	if meta.Mode != mode.ChatCompletions {
		return do.ConvertRequest(meta, req)
	}

	// Load plugin configuration
	pluginConfig := Config{}
	if err := meta.ModelConfig.LoadPluginConfig("web-search", &pluginConfig); err != nil {
		return do.ConvertRequest(meta, req)
	}

	// Skip if plugin is disabled
	if !pluginConfig.EnablePlugin {
		return do.ConvertRequest(meta, req)
	}

	// Apply default configuration values if needed
	if err := p.validateAndApplyDefaults(&pluginConfig); err != nil {
		return do.ConvertRequest(meta, req)
	}

	// Initialize search engines
	engines, arxivExists, err := p.initializeSearchEngines(pluginConfig.SearchFrom)
	if err != nil || len(engines) == 0 {
		return do.ConvertRequest(meta, req)
	}

	// Read and parse request body
	body, err := common.GetRequestBody(req)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	var chatRequest map[string]any
	if err := sonic.Unmarshal(body, &chatRequest); err != nil {
		return do.ConvertRequest(meta, req)
	}

	// Check if web search should be enabled for this request
	webSearchOptions, hasWebSearchOptions := chatRequest["web_search_options"].(map[string]any)
	if !pluginConfig.DefaultEnable && !hasWebSearchOptions {
		return do.ConvertRequest(meta, req)
	}

	// Extract user query from messages
	messages, ok := chatRequest["messages"].([]any)
	if !ok || len(messages) == 0 {
		return do.ConvertRequest(meta, req)
	}

	queryIndex, query := p.extractUserQuery(messages)
	if query == "" {
		return do.ConvertRequest(meta, req)
	}

	// Prepare search rewrite prompt if configured
	searchRewritePrompt := p.prepareSearchRewritePrompt(pluginConfig.SearchRewrite, arxivExists, webSearchOptions)

	// Generate search contexts
	searchContexts := p.generateSearchContexts(meta, pluginConfig, query, searchRewritePrompt)
	if len(searchContexts) == 0 {
		return nil, errors.New("no valid search contexts found")
	}

	// Execute searches
	searchResults, err := p.executeSearches(context.Background(), engines, searchContexts)
	if err != nil || len(searchResults) == 0 {
		return do.ConvertRequest(meta, req)
	}

	// Format search results and modify request
	modifiedRequest, references := p.formatSearchResults(chatRequest, queryIndex, query, searchResults, pluginConfig)

	delete(modifiedRequest, "web_search_options")

	// Create new request body
	modifiedBody, err := sonic.Marshal(modifiedRequest)
	if err != nil {
		return do.ConvertRequest(meta, req)
	}

	// Update the request
	common.SetRequestBody(req, modifiedBody)
	defer common.SetRequestBody(req, body)

	// Store references in context if needed
	if pluginConfig.NeedReference && references != "" {
		meta.Set("references", references)
	}

	return do.ConvertRequest(meta, req)
}

// validateAndApplyDefaults validates configuration and applies default values
func (p *WebSearch) validateAndApplyDefaults(config *Config) error {
	// Set default max results
	if config.MaxResults == 0 {
		config.MaxResults = 10
	}

	// Configure reference settings
	if config.NeedReference {
		if config.ReferenceLocation == "" {
			config.ReferenceLocation = "head"
		} else if config.ReferenceLocation != "head" && config.ReferenceLocation != "tail" {
			return errors.New("invalid reference location")
		}

		if config.ReferenceFormat == "" {
			config.ReferenceFormat = "**References:**\n%s"
		} else if !strings.Contains(config.ReferenceFormat, "%s") {
			return errors.New("invalid reference format")
		}
	}

	// Set default prompt template if not provided
	if config.PromptTemplate == "" {
		if config.NeedReference {
			config.PromptTemplate = `# 以下内容是基于用户发送的消息的搜索结果:
{search_results}
在我给你的搜索结果中，每个结果都是[webpage X begin]...[webpage X end]格式的，X代表每篇文章的数字索引。请在适当的情况下在句子末尾引用上下文。请按照引用编号[X]的格式在答案中对应部分引用上下文。如果一句话源自多个上下文，请列出所有相关的引用编号，例如[3][5]，切记不要将引用集中在最后返回引用编号，而是在答案对应部分列出。
在回答时，请注意以下几点：
- 今天是北京时间：{cur_date}。
- 并非搜索结果的所有内容都与用户的问题密切相关，你需要结合问题，对搜索结果进行甄别、筛选。
- 对于列举类的问题（如列举所有航班信息），尽量将答案控制在10个要点以内，并告诉用户可以查看搜索来源、获得完整信息。优先提供信息完整、最相关的列举项；如非必要，不要主动告诉用户搜索结果未提供的内容。
- 对于创作类的问题（如写论文），请务必在正文的段落中引用对应的参考编号，例如[3][5]，不能只在文章末尾引用。你需要解读并概括用户的题目要求，选择合适的格式，充分利用搜索结果并抽取重要信息，生成符合用户要求、极具思想深度、富有创造力与专业性的答案。你的创作篇幅需要尽可能延长，对于每一个要点的论述要推测用户的意图，给出尽可能多角度的回答要点，且务必信息量大、论述详尽。
- 如果回答很长，请尽量结构化、分段落总结。如果需要分点作答，尽量控制在5个点以内，并合并相关的内容。
- 对于客观类的问答，如果问题的答案非常简短，可以适当补充一到两句相关信息，以丰富内容。
- 你需要根据用户要求和回答内容选择合适、美观的回答格式，确保可读性强。
- 你的回答应该综合多个相关网页来回答，不能重复引用一个网页。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
		} else {
			config.PromptTemplate = `# 以下内容是基于用户发送的消息的搜索结果:
{search_results}
在我给你的搜索结果中，每个结果都是[webpage begin]...[webpage end]格式的。
在回答时，请注意以下几点：
- 今天是北京时间：{cur_date}。
- 并非搜索结果的所有内容都与用户的问题密切相关，你需要结合问题，对搜索结果进行甄别、筛选。
- 对于列举类的问题（如列举所有航班信息），尽量将答案控制在10个要点以内。如非必要，不要主动告诉用户搜索结果未提供的内容。
- 对于创作类的问题（如写论文），你需要解读并概括用户的题目要求，选择合适的格式，充分利用搜索结果并抽取重要信息，生成符合用户要求、极具思想深度、富有创造力与专业性的答案。你的创作篇幅需要尽可能延长，对于每一个要点的论述要推测用户的意图，给出尽可能多角度的回答要点，且务必信息量大、论述详尽。
- 如果回答很长，请尽量结构化、分段落总结。如果需要分点作答，尽量控制在5个点以内，并合并相关的内容。
- 对于客观类的问答，如果问题的答案非常简短，可以适当补充一到两句相关信息，以丰富内容。
- 你需要根据用户要求和回答内容选择合适、美观的回答格式，确保可读性强。
- 你的回答应该综合多个相关网页来回答，但回答中不要给出网页的引用来源。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
		}
	}

	// Validate prompt template
	if !strings.Contains(config.PromptTemplate, "{search_results}") ||
		!strings.Contains(config.PromptTemplate, "{question}") {
		return errors.New("invalid prompt template")
	}

	return nil
}

// initializeSearchEngines creates search engine instances based on configuration
func (p *WebSearch) initializeSearchEngines(configs []EngineConfig) ([]engine.Engine, bool, error) {
	var engines []engine.Engine
	var arxivExists bool

	for _, e := range configs {
		switch e.Type {
		case "bing":
			var spec BingSpec
			if err := e.LoadSpec(&spec); err != nil {
				return nil, false, err
			}
			engines = append(engines, engine.NewBingEngine(spec.APIKey))
		case "google":
			var spec GoogleSpec
			if err := e.LoadSpec(&spec); err != nil {
				return nil, false, err
			}
			engines = append(engines, engine.NewGoogleEngine(spec.APIKey, spec.CX))
		case "arxiv":
			engines = append(engines, engine.NewArxivEngine())
			arxivExists = true
		default:
			return nil, false, fmt.Errorf("unsupported engine type: %s", e.Type)
		}
	}

	return engines, arxivExists, nil
}

// extractUserQuery finds the last user message in the conversation
func (p *WebSearch) extractUserQuery(messages []any) (int, string) {
	for i := len(messages) - 1; i >= 0; i-- {
		msg, ok := messages[i].(map[string]any)
		if !ok {
			continue
		}

		if role, ok := msg["role"].(string); ok && role == "user" {
			if content, ok := msg["content"].(string); ok {
				return i, content
			}
			return i, ""
		}
	}
	return -1, ""
}

// prepareSearchRewritePrompt prepares the prompt for search query rewriting
func (p *WebSearch) prepareSearchRewritePrompt(searchRewrite SearchRewrite, arxivExists bool, webSearchOptions map[string]any) string {
	if !searchRewrite.Enable {
		return ""
	}

	// Select appropriate prompt template
	var searchRewritePromptTemplate string
	if arxivExists {
		searchRewritePromptTemplate = arxivSearchPrompts
	} else {
		searchRewritePromptTemplate = internetSearchPrompts
	}

	// Adjust max count based on search context size if specified
	maxCount := searchRewrite.MaxCount
	if webSearchOptions != nil {
		if searchContextSize, ok := webSearchOptions["search_context_size"].(string); ok {
			switch searchContextSize {
			case "low":
				maxCount = 1
			case "medium":
				maxCount = 3
			case "high":
				maxCount = 5
			}
		}
	}

	// Replace placeholder with actual max count
	return strings.ReplaceAll(searchRewritePromptTemplate, "{max_count}", strconv.Itoa(maxCount))
}

// generateSearchContexts creates search contexts based on the user query
func (p *WebSearch) generateSearchContexts(m *meta.Meta, config Config, query string, searchRewritePrompt string) []engine.SearchQuery {
	if searchRewritePrompt == "" {
		return []engine.SearchQuery{{
			Queries:  []string{query},
			Language: config.DefaultLanguage,
		}}
	}

	rewriteBody, err := sonic.Marshal(map[string]any{
		"stream":     false,
		"max_tokens": 4096,
		"model":      config.SearchRewrite.ModelName,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": strings.ReplaceAll(searchRewritePrompt, "{question}", query),
			},
		},
	})
	if err != nil {
		return nil
	}

	w := httptest.NewRecorder()
	newc, _ := gin.CreateTestContext(w)
	newc.Request = &http.Request{
		URL:    &url.URL{},
		Body:   io.NopCloser(bytes.NewReader(rewriteBody)),
		Header: make(http.Header),
	}
	middleware.SetRequestID(newc, "web-search-rewrite")

	modelName := config.SearchRewrite.ModelName
	if modelName == "" {
		modelName = m.OriginModel
	}
	newMeta := meta.NewMeta(
		nil,
		mode.ChatCompletions,
		modelName,
		model.ModelConfig{
			Model: modelName,
			Type:  mode.ChatCompletions,
		},
		meta.WithRequestID("web-search-rewrite"),
	)
	if config.SearchRewrite.ModelName == "" {
		newMeta.CopyChannelFromMeta(m)
	} else {
		channel, err := p.GetChannel(config.SearchRewrite.ModelName)
		if err != nil {
			return nil
		}
		newMeta.SetChannel(channel)
	}
	adaptor, ok := adaptors.GetAdaptor(newMeta.Channel.Type)
	if !ok {
		return nil
	}
	controller.Handle(adaptor, newc, newMeta)

	contentNode, err := sonic.Get(w.Body.Bytes(), "choices", 0, "message", "content")
	if err != nil {
		return nil
	}

	content, err := contentNode.String()
	if err != nil || content == "" {
		return nil
	}

	if strings.Contains(content, "none") {
		return nil
	}

	// Parse search queries from LLM response
	var searchContexts []engine.SearchQuery
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		engineType := strings.TrimSpace(parts[0])
		queryStr := strings.TrimSpace(parts[1])

		var ctx engine.SearchQuery
		ctx.Language = config.DefaultLanguage

		switch {
		case engineType == "internet":
			ctx.Queries = []string{queryStr}
		default:
			// Arxiv category
			ctx.ArxivCategory = engineType
			ctx.Queries = strings.Split(queryStr, ",")
			for i := range ctx.Queries {
				ctx.Queries[i] = strings.TrimSpace(ctx.Queries[i])
			}
		}

		if len(ctx.Queries) > 0 {
			searchContexts = append(searchContexts, ctx)
			if ctx.ArxivCategory != "" {
				// Conduct inquiries in all areas to increase recall.
				backupCtx := ctx
				backupCtx.ArxivCategory = ""
				searchContexts = append(searchContexts, backupCtx)
			}
		}
	}
	return searchContexts
}

// executeSearches performs searches using all configured engines
func (p *WebSearch) executeSearches(ctx context.Context, engines []engine.Engine, searchContexts []engine.SearchQuery) ([]engine.SearchResult, error) {
	var allResults []engine.SearchResult
	resultsChan := make(chan []engine.SearchResult, len(engines)*len(searchContexts))
	errorsChan := make(chan error, len(engines)*len(searchContexts))

	searchCount := 0
	for _, eng := range engines {
		for _, searchCtx := range searchContexts {
			searchCount++
			go func(e engine.Engine, sc engine.SearchQuery) {
				results, err := e.Search(ctx, engine.SearchQuery{
					Queries:       sc.Queries,
					MaxResults:    10,
					Language:      sc.Language,
					ArxivCategory: sc.ArxivCategory,
				})
				if err != nil {
					errorsChan <- err
					return
				}
				resultsChan <- results
			}(eng, searchCtx)
		}
	}

	// Collect results with timeout
	timeout := time.After(10 * time.Second)
	received := 0
	for received < searchCount {
		select {
		case results := <-resultsChan:
			allResults = append(allResults, results...)
			received++
		case <-errorsChan:
			received++
		case <-timeout:
			return allResults, errors.New("search timeout")
		}
	}

	// Deduplicate results by link
	seen := make(map[string]bool)
	var uniqueResults []engine.SearchResult
	for _, result := range allResults {
		if !seen[result.Link] {
			seen[result.Link] = true
			uniqueResults = append(uniqueResults, result)
		}
	}

	return uniqueResults, nil
}

// formatSearchResults formats search results for the prompt
func (p *WebSearch) formatSearchResults(chatRequest map[string]any, queryIndex int, query string, searchResults []engine.SearchResult, config Config) (map[string]any, string) {
	var formattedResults []string
	var formattedReferences []string

	for i, result := range searchResults {
		if config.NeedReference {
			formattedResults = append(formattedResults,
				fmt.Sprintf("[webpage %d begin]\n%s\n[webpage %d end]", i+1, result.Content, i+1))
			formattedReferences = append(formattedReferences,
				fmt.Sprintf("[%d] [%s](%s)", i+1, result.Title, result.Link))
		} else {
			formattedResults = append(formattedResults,
				fmt.Sprintf("[webpage begin]\n%s\n[webpage end]", result.Content))
		}
	}

	// Fill template
	curDate := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006年1月2日")
	searchResultsStr := strings.Join(formattedResults, "\n")

	prompt := strings.Replace(config.PromptTemplate, "{search_results}", searchResultsStr, 1)
	prompt = strings.Replace(prompt, "{question}", query, 1)
	prompt = strings.Replace(prompt, "{cur_date}", curDate, 1)

	// Update message
	messages := chatRequest["messages"].([]any)
	messages[queryIndex].(map[string]any)["content"] = prompt
	chatRequest["messages"] = messages

	references := ""
	if config.NeedReference {
		references = strings.Join(formattedReferences, "\n\n")
	}

	return chatRequest, references
}

type responseWriter struct {
	gin.ResponseWriter
	writed     bool
	references string
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.writed {
		return rw.ResponseWriter.Write(b)
	}
	node, err := sonic.Get(b)
	if err != nil {
		return rw.ResponseWriter.Write(b)
	}
	var contentNode *ast.Node
	if utils.IsStreamResponseWithHeader(rw.ResponseWriter.Header()) {
		contentNode = node.GetByPath("choices", 0, "delta", "content")
	} else {
		contentNode = node.GetByPath("choices", 0, "message", "content")
	}
	content, err := contentNode.String()
	if err != nil {
		return 0, err
	}
	refContent := fmt.Sprintf("%s\n\n%s", rw.references, content)
	*contentNode = ast.NewString(refContent)
	b, err = sonic.Marshal(&node)
	if err != nil {
		return 0, err
	}
	rw.writed = true
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteString(s string) (int, error) {
	if rw.writed {
		return rw.ResponseWriter.WriteString(s)
	}
	return rw.ResponseWriter.WriteString(s)
}

// DoResponse handles response modification for references
func (p *WebSearch) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response, do adaptor.DoResponse) (*model.Usage, adaptor.Error) {
	references := meta.GetString("references")
	if references == "" {
		return do.DoResponse(meta, c, resp)
	}
	rw := &responseWriter{
		ResponseWriter: c.Writer,
		references:     references,
	}
	c.Writer = rw
	defer func() {
		c.Writer = rw.ResponseWriter
	}()
	return do.DoResponse(meta, c, resp)
}
