package llm

import (
	"context"
	"fmt"

	"ansmeee-ai-agent/internal/config"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// Provider wraps a langchaingo LLM model.
type Provider struct {
	model llms.Model
	cfg   *config.LLMConfig
}

// New creates a new LLM provider from configuration.
func New(cfg *config.LLMConfig) (*Provider, error) {
	return NewWithOverride(cfg, "", "", "")
}

// NewWithOverride creates an LLM provider with optional model/base_url/token overrides.
func NewWithOverride(cfg *config.LLMConfig, model, baseURL, token string) (*Provider, error) {
	if model == "" {
		model = cfg.Model
	}
	if baseURL == "" {
		baseURL = cfg.BaseURL
	}
	if token == "" {
		token = cfg.APIKey
	}

	llmModel, err := openai.New(
		openai.WithToken(token),
		openai.WithModel(model),
		openai.WithBaseURL(baseURL),
	)
	if err != nil {
		return nil, fmt.Errorf("create LLM model: %w", err)
	}

	return &Provider{model: llmModel, cfg: cfg}, nil
}

// ChatResult is the full result of a chat call.
type ChatResult struct {
	Content   string
	ToolCalls []llms.ToolCall
}

// Chat sends messages and returns the full result including any tool calls.
func (p *Provider) Chat(ctx context.Context, messages []MessageContent, toolList []tools.Tool) (*ChatResult, error) {
	msgs := toLLMMessages(messages)
	opts := []llms.CallOption{
		llms.WithTemperature(p.cfg.Temperature),
		llms.WithMaxTokens(p.cfg.MaxTokens),
	}
	if len(toolList) > 0 {
		opts = append(opts, llms.WithTools(toLLMTools(toolList)))
	}

	resp, err := p.model.GenerateContent(ctx, msgs, opts...)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from LLM")
	}

	choice := resp.Choices[0]
	return &ChatResult{
		Content:   choice.Content,
		ToolCalls: choice.ToolCalls,
	}, nil
}

// StreamChat sends messages and returns a streaming channel.
func (p *Provider) StreamChat(ctx context.Context, messages []MessageContent) (<-chan string, <-chan error) {
	textCh := make(chan string, 10)
	errCh := make(chan error, 1)

	msgs := toLLMMessages(messages)

	go func() {
		defer close(textCh)
		defer close(errCh)

		_, err := p.model.GenerateContent(ctx, msgs,
			llms.WithTemperature(p.cfg.Temperature),
			llms.WithMaxTokens(p.cfg.MaxTokens),
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				select {
				case textCh <- string(chunk):
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}),
		)
		if err != nil {
			errCh <- err
		}
	}()

	return textCh, errCh
}

// MessageContent is an internal message type.
type MessageContent struct {
	Role         Role
	Content      string
	ToolCallID   string
	ToolCallName string
	ToolCalls    []llms.ToolCall
}

// Role represents a message role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "human"
	RoleAssistant Role = "ai"
	RoleTool      Role = "tool"
)

func toLLMMessages(messages []MessageContent) []llms.MessageContent {
	result := make([]llms.MessageContent, len(messages))
	for i, m := range messages {
		var parts []llms.ContentPart
		if m.Content != "" {
			parts = append(parts, llms.TextPart(m.Content))
		}
		for _, tc := range m.ToolCalls {
			parts = append(parts, tc)
		}
		if m.ToolCallID != "" {
			parts = append(parts, llms.ToolCallResponse{
				ToolCallID: m.ToolCallID,
				Name:       m.ToolCallName,
				Content:    m.Content,
			})
		}

		result[i] = llms.MessageContent{
			Role:  llms.ChatMessageType(m.Role),
			Parts: parts,
		}
	}
	return result
}

func toLLMTools(toolList []tools.Tool) []llms.Tool {
	result := make([]llms.Tool, len(toolList))
	for i, t := range toolList {
		result[i] = llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        t.Name(),
				Description: t.Description(),
			},
		}
	}
	return result
}

// WithOverride creates a new Provider with the given overrides.
func (p *Provider) WithOverride(model, baseURL, token string) (*Provider, error) {
	return NewWithOverride(p.cfg, model, baseURL, token)
}
