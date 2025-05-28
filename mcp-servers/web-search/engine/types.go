package engine

import (
	"context"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title   string
	Link    string
	Content string
}

// SearchQuery represents search parameters
type SearchQuery struct {
	Query         string
	MaxResults    int
	Language      string
	ArxivCategory string
}

// Engine interface for search engines
type Engine interface {
	Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)
}
