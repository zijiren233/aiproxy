package firecrawl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

// FirecrawlClient represents the Firecrawl API client
type FirecrawlClient struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// ScrapeParams represents parameters for scraping
type ScrapeParams struct {
	URL                 string          `json:"url"`
	Formats             []string        `json:"formats,omitempty"`
	OnlyMainContent     *bool           `json:"onlyMainContent,omitempty"`
	IncludeTags         []string        `json:"includeTags,omitempty"`
	ExcludeTags         []string        `json:"excludeTags,omitempty"`
	WaitFor             *int            `json:"waitFor,omitempty"`
	Timeout             *int            `json:"timeout,omitempty"`
	Actions             []Action        `json:"actions,omitempty"`
	Extract             *ExtractConfig  `json:"extract,omitempty"`
	Mobile              *bool           `json:"mobile,omitempty"`
	SkipTLSVerification *bool           `json:"skipTlsVerification,omitempty"`
	RemoveBase64Images  *bool           `json:"removeBase64Images,omitempty"`
	Location            *LocationConfig `json:"location,omitempty"`
	Origin              string          `json:"origin,omitempty"`
}

// Action represents an action to perform before scraping
type Action struct {
	Type         string `json:"type"`
	Selector     string `json:"selector,omitempty"`
	Milliseconds *int   `json:"milliseconds,omitempty"`
	Text         string `json:"text,omitempty"`
	Key          string `json:"key,omitempty"`
	Direction    string `json:"direction,omitempty"`
	Script       string `json:"script,omitempty"`
	FullPage     *bool  `json:"fullPage,omitempty"`
}

// ExtractConfig represents extraction configuration
type ExtractConfig struct {
	Schema       map[string]any `json:"schema,omitempty"`
	SystemPrompt string         `json:"systemPrompt,omitempty"`
	Prompt       string         `json:"prompt,omitempty"`
}

// LocationConfig represents location settings
type LocationConfig struct {
	Country   string   `json:"country,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

// MapParams represents parameters for mapping
type MapParams struct {
	URL               string `json:"url"`
	Search            string `json:"search,omitempty"`
	IgnoreSitemap     *bool  `json:"ignoreSitemap,omitempty"`
	SitemapOnly       *bool  `json:"sitemapOnly,omitempty"`
	IncludeSubdomains *bool  `json:"includeSubdomains,omitempty"`
	Limit             *int   `json:"limit,omitempty"`
	Origin            string `json:"origin,omitempty"`
}

// CrawlParams represents parameters for crawling
type CrawlParams struct {
	URL                    string        `json:"url"`
	ExcludePaths           []string      `json:"excludePaths,omitempty"`
	IncludePaths           []string      `json:"includePaths,omitempty"`
	MaxDepth               *int          `json:"maxDepth,omitempty"`
	IgnoreSitemap          *bool         `json:"ignoreSitemap,omitempty"`
	Limit                  *int          `json:"limit,omitempty"`
	AllowBackwardLinks     *bool         `json:"allowBackwardLinks,omitempty"`
	AllowExternalLinks     *bool         `json:"allowExternalLinks,omitempty"`
	Webhook                any           `json:"webhook,omitempty"`
	DeduplicateSimilarURLs *bool         `json:"deduplicateSimilarURLs,omitempty"`
	IgnoreQueryParameters  *bool         `json:"ignoreQueryParameters,omitempty"`
	ScrapeOptions          *ScrapeParams `json:"scrapeOptions,omitempty"`
	Origin                 string        `json:"origin,omitempty"`
}

// SearchParams represents parameters for searching
type SearchParams struct {
	Query         string          `json:"query"`
	Limit         *int            `json:"limit,omitempty"`
	Lang          string          `json:"lang,omitempty"`
	Country       string          `json:"country,omitempty"`
	TBS           string          `json:"tbs,omitempty"`
	Filter        string          `json:"filter,omitempty"`
	Location      *LocationConfig `json:"location,omitempty"`
	ScrapeOptions *ScrapeConfig   `json:"scrapeOptions,omitempty"`
	Origin        string          `json:"origin,omitempty"`
}

// ScrapeConfig represents scrape configuration for search
type ScrapeConfig struct {
	Formats         []string `json:"formats,omitempty"`
	OnlyMainContent *bool    `json:"onlyMainContent,omitempty"`
	WaitFor         *int     `json:"waitFor,omitempty"`
	IncludeTags     []string `json:"includeTags,omitempty"`
	ExcludeTags     []string `json:"excludeTags,omitempty"`
	Timeout         *int     `json:"timeout,omitempty"`
}

// ExtractParams represents parameters for extraction
type ExtractParams struct {
	URLs               []string       `json:"urls"`
	Prompt             string         `json:"prompt,omitempty"`
	SystemPrompt       string         `json:"systemPrompt,omitempty"`
	Schema             map[string]any `json:"schema,omitempty"`
	AllowExternalLinks *bool          `json:"allowExternalLinks,omitempty"`
	EnableWebSearch    *bool          `json:"enableWebSearch,omitempty"`
	IncludeSubdomains  *bool          `json:"includeSubdomains,omitempty"`
	Origin             string         `json:"origin,omitempty"`
}

// DeepResearchParams represents parameters for deep research
type DeepResearchParams struct {
	Query     string `json:"query"`
	MaxDepth  *int   `json:"maxDepth,omitempty"`
	TimeLimit *int   `json:"timeLimit,omitempty"`
	MaxURLs   *int   `json:"maxUrls,omitempty"`
	Origin    string `json:"origin,omitempty"`
}

// GenerateLLMsTextParams represents parameters for LLMs.txt generation
type GenerateLLMsTextParams struct {
	URL          string `json:"url"`
	MaxURLs      *int   `json:"maxUrls,omitempty"`
	ShowFullText *bool  `json:"showFullText,omitempty"`
	Origin       string `json:"origin,omitempty"`
}

// Response types
type ScrapeResponse struct {
	Success    bool               `json:"success"`
	Data       *FirecrawlDocument `json:"data,omitempty"`
	Markdown   string             `json:"markdown,omitempty"`
	HTML       string             `json:"html,omitempty"`
	RawHTML    string             `json:"rawHtml,omitempty"`
	Links      []string           `json:"links,omitempty"`
	Screenshot string             `json:"screenshot,omitempty"`
	Extract    map[string]any     `json:"extract,omitempty"`
	Warning    string             `json:"warning,omitempty"`
	Error      string             `json:"error,omitempty"`
}

type MapResponse struct {
	Success bool     `json:"success"`
	Links   []string `json:"links,omitempty"`
	Error   string   `json:"error,omitempty"`
}

type CrawlResponse struct {
	Success bool   `json:"success"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error,omitempty"`
}

type CrawlStatusResponse struct {
	Success     bool                `json:"success"`
	Status      string              `json:"status"`
	Completed   int                 `json:"completed"`
	Total       int                 `json:"total"`
	CreditsUsed int                 `json:"creditsUsed"`
	ExpiresAt   string              `json:"expiresAt"`
	Data        []FirecrawlDocument `json:"data"`
	Error       string              `json:"error,omitempty"`
}

type SearchResponse struct {
	Success bool                `json:"success"`
	Data    []FirecrawlDocument `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}

type ExtractResponse struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Warning string         `json:"warning,omitempty"`
	Error   string         `json:"error,omitempty"`
}

type DeepResearchResponse struct {
	Success bool              `json:"success"`
	Data    *DeepResearchData `json:"data,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type DeepResearchData struct {
	FinalAnalysis string           `json:"finalAnalysis"`
	Activities    []map[string]any `json:"activities"`
	Sources       []map[string]any `json:"sources"`
}

type GenerateLLMsTextResponse struct {
	Success bool          `json:"success"`
	Data    *LLMsTextData `json:"data,omitempty"`
	Error   string        `json:"error,omitempty"`
}

type LLMsTextData struct {
	LLMsText     string `json:"llmstxt"`
	LLMsFullText string `json:"llmsfulltxt,omitempty"`
}

type FirecrawlDocument struct {
	URL         string         `json:"url,omitempty"`
	Markdown    string         `json:"markdown,omitempty"`
	HTML        string         `json:"html,omitempty"`
	RawHTML     string         `json:"rawHtml,omitempty"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// NewFirecrawlClient creates a new Firecrawl client
func NewFirecrawlClient(apiKey, apiURL string) *FirecrawlClient {
	return &FirecrawlClient{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// makeRequest makes an HTTP request to the Firecrawl API
func (c *FirecrawlClient) makeRequest(
	ctx context.Context,
	method, endpoint string,
	body any,
) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := sonic.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ScrapeURL scrapes a single URL
func (c *FirecrawlClient) ScrapeURL(
	ctx context.Context,
	params ScrapeParams,
) (*ScrapeResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/scrape", params)
	if err != nil {
		return nil, err
	}

	var response ScrapeResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// MapURL maps a website to discover URLs
func (c *FirecrawlClient) MapURL(ctx context.Context, params MapParams) (*MapResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/map", params)
	if err != nil {
		return nil, err
	}

	var response MapResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// AsyncCrawlURL starts an asynchronous crawl
func (c *FirecrawlClient) AsyncCrawlURL(
	ctx context.Context,
	params CrawlParams,
) (*CrawlResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/crawl", params)
	if err != nil {
		return nil, err
	}

	var response CrawlResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CheckCrawlStatus checks the status of a crawl job
func (c *FirecrawlClient) CheckCrawlStatus(
	ctx context.Context,
	id string,
) (*CrawlStatusResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodGet, "/v1/crawl/"+id, nil)
	if err != nil {
		return nil, err
	}

	var response CrawlStatusResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// Search searches the web
func (c *FirecrawlClient) Search(
	ctx context.Context,
	params SearchParams,
) (*SearchResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/search", params)
	if err != nil {
		return nil, err
	}

	var response SearchResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// Extract extracts structured data
func (c *FirecrawlClient) Extract(
	ctx context.Context,
	params ExtractParams,
) (*ExtractResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/extract", params)
	if err != nil {
		return nil, err
	}

	var response ExtractResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeepResearch conducts deep web research
func (c *FirecrawlClient) DeepResearch(
	ctx context.Context,
	params DeepResearchParams,
) (*DeepResearchResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/research", params)
	if err != nil {
		return nil, err
	}

	var response DeepResearchResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GenerateLLMsText generates LLMs.txt file
func (c *FirecrawlClient) GenerateLLMsText(
	ctx context.Context,
	params GenerateLLMsTextParams,
) (*GenerateLLMsTextResponse, error) {
	respBody, err := c.makeRequest(ctx, http.MethodPost, "/v1/llmstxt", params)
	if err != nil {
		return nil, err
	}

	var response GenerateLLMsTextResponse
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
