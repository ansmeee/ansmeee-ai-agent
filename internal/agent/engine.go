package agent

import (
	"context"
	"fmt"
	"strings"

	"ansmeee-ai-agent/internal/llm"
	"ansmeee-ai-agent/internal/memory"
	"ansmeee-ai-agent/internal/models"
	"ansmeee-ai-agent/internal/tool"
)

const defaultSystemPrompt = "直接回答用户问题。当需要获取实时信息或进行数学计算时使用工具。评估工具结果后再回复。不要做自我介绍。"

// Engine orchestrates LLM calls with tools and memory.
type Engine struct {
	llm          *llm.Provider
	registry     *tool.Registry
	mem          memory.SessionStore
	callback     *Callback
	systemPrompt string
	maxIter      int
}

// EngineOption configures the engine.
type EngineOption func(*Engine)

// WithSystemPrompt sets a custom system prompt.
func WithSystemPrompt(prompt string) EngineOption {
	return func(e *Engine) { e.systemPrompt = prompt }
}

// WithMaxIter sets the maximum tool-calling iterations.
func WithMaxIter(n int) EngineOption {
	return func(e *Engine) { e.maxIter = n }
}

// New creates a new Agent engine.
func New(llmProvider *llm.Provider, reg *tool.Registry, mem memory.SessionStore, cb *Callback, opts ...EngineOption) *Engine {
	e := &Engine{
		llm:          llmProvider,
		registry:     reg,
		mem:          mem,
		callback:     cb,
		systemPrompt: defaultSystemPrompt,
		maxIter:      5,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// ToolCallEvent records a tool invocation.
type ToolCallEvent struct {
	ToolName string `json:"tool_name"`
	Input    string `json:"input"`
	Output   string `json:"output"`
}

// StreamEvent is emitted during streaming.
type StreamEvent struct {
	Type    string `json:"type"` // "chunk", "done", "error"
	Content string `json:"content"`
	Error   error  `json:"-"`
}

// ProcessStream handles a streaming chat turn.
func (e *Engine) ProcessStream(ctx context.Context, sessionID, userMessage, promptOverride string, modelCfg *models.ModelConfig, userID int64) <-chan StreamEvent {
	ch := make(chan StreamEvent, 20)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("panic: %v", r)}
			}
		}()

		prompt := e.systemPrompt
		if promptOverride != "" {
			prompt = promptOverride
		}

		if err := e.mem.AddMessage(ctx, sessionID, memory.Message{Role: "user", Content: userMessage}, userID); err != nil {
			ch <- StreamEvent{Type: "error", Error: fmt.Errorf("save user message: %w", err)}
			return
		}

		history, _ := e.mem.History(ctx, sessionID)
		messages := buildMessages(prompt, history)

		e.callback.OnLLMStart(ctx, sessionID)

		llmProvider := e.llm
		if modelCfg != nil && modelCfg.Model != "" {
			if p, err := e.llm.WithOverride(modelCfg.Model, modelCfg.BaseURL, modelCfg.Token); err == nil {
				llmProvider = p
			}
		}
		textCh, errCh := llmProvider.StreamChat(ctx, messages)
		var full strings.Builder

	loop:
		for {
			select {
			case chunk, ok := <-textCh:
				if !ok {
					break loop
				}
				full.WriteString(chunk)
				ch <- StreamEvent{Type: "chunk", Content: chunk}
			case err, ok := <-errCh:
				if ok && err != nil {
					ch <- StreamEvent{Type: "error", Error: err}
					return
				}
				break loop
			case <-ctx.Done():
				ch <- StreamEvent{Type: "error", Error: ctx.Err()}
				return
			}
		}

		fullText := full.String()
		if err := e.mem.AddMessage(ctx, sessionID, memory.Message{Role: "assistant", Content: fullText}, userID); err != nil {
			ch <- StreamEvent{Type: "error", Error: fmt.Errorf("save assistant message: %w", err)}
			return
		}

		e.callback.OnLLMEnd(ctx, sessionID, 0, 0)
		ch <- StreamEvent{Type: "done", Content: sessionID}
	}()

	return ch
}

func buildMessages(systemPrompt string, history []memory.Message) []llm.MessageContent {
	msgs := []llm.MessageContent{{Role: llm.RoleSystem, Content: systemPrompt}}
	for _, m := range history {
		role := llm.RoleUser
		switch m.Role {
		case "assistant":
			role = llm.RoleAssistant
		case "system":
			role = llm.RoleSystem
		}
		msgs = append(msgs, llm.MessageContent{
			Role:    role,
			Content: m.Content,
		})
	}
	return msgs
}
