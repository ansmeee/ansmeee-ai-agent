package tool

import (
	"context"
	"fmt"
)

// WebSearch provides a placeholder for web search capability.
type WebSearch struct{}

// Name returns the tool name.
func (w *WebSearch) Name() string { return "web_search" }

// Description returns the tool description.
func (w *WebSearch) Description() string {
	return "Search the web for information. Provide a search query string."
}

// Parameters returns the JSON Schema for the tool's input.
func (w *WebSearch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query string",
			},
		},
		"required": []string{"query"},
	}
}

// Call is a placeholder that indicates the search query was received.
func (w *WebSearch) Call(ctx context.Context, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("search query is required")
	}
	return fmt.Sprintf("Web search for %q is not configured. Please set up an API key for web search.", input), nil
}
