package llm

import (
	"ansmeee-ai-agent/internal/config"
	"github.com/tmc/langchaingo/llms"
)

// NewForTest creates a Provider backed by a custom llms.Model, for use in tests.
func NewForTest(model llms.Model) *Provider {
	return &Provider{
		model: model,
		cfg: &config.LLMConfig{
			Model:       "test-model",
			Temperature: 0.7,
			MaxTokens:   1024,
		},
	}
}
