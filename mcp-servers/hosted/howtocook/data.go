package howtocook

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

// fetchRecipes fetches recipes from remote URL
func (s *Server) fetchRecipes(ctx context.Context) ([]Recipe, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, RecipesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var recipes []Recipe
	if err := sonic.Unmarshal(body, &recipes); err != nil {
		return nil, fmt.Errorf("failed to parse recipes: %w", err)
	}

	return recipes, nil
}

// getAllCategories returns all unique categories from recipes
func (s *Server) getAllCategories() []string {
	categorySet := make(map[string]bool)
	for _, recipe := range s.recipes {
		if recipe.Category != "" {
			categorySet[recipe.Category] = true
		}
	}

	categories := make([]string, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}

	return categories
}
