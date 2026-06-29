package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"ansmeee-ai-agent/internal/llm"
	"ansmeee-ai-agent/internal/memory"
	"ansmeee-ai-agent/internal/models"
	"ansmeee-ai-agent/internal/tool"
	"ansmeee-ai-agent/internal/tracing"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const defaultSystemPrompt = "直接回答用户问题。当需要获取实时信息或进行数学计算时使用工具。评估工具结果后再回复。不要做自我介绍。"

const maxParallelTools = 5

// Engine orchestrates LLM calls with tools and memory.
type Engine struct {
	llm                *llm.Provider
	registry           *tool.Registry
	mem                memory.SessionStore
	callback           *Callback
	systemPrompt       string
	maxIter            int
	toolTimeout        time.Duration
	maxOutputLength    int
	parallelToolCalls  bool
	maxContextMessages int
}

// EngineOption configures the engine.
type EngineOption func(*Engine)

func WithSystemPrompt(prompt string) EngineOption {
	return func(e *Engine) { e.systemPrompt = prompt }
}

func WithMaxIter(n int) EngineOption {
	return func(e *Engine) { e.maxIter = n }
}

func WithToolTimeout(d time.Duration) EngineOption {
	return func(e *Engine) { e.toolTimeout = d }
}

func WithMaxOutputLength(n int) EngineOption {
	return func(e *Engine) { e.maxOutputLength = n }
}

func WithParallelToolCalls(b bool) EngineOption {
	return func(e *Engine) { e.parallelToolCalls = b }
}

func WithMaxContextMessages(n int) EngineOption {
	return func(e *Engine) { e.maxContextMessages = n }
}

// New creates a new Agent engine.
func New(llmProvider *llm.Provider, reg *tool.Registry, mem memory.SessionStore, cb *Callback, opts ...EngineOption) *Engine {
	e := &Engine{
		llm:                llmProvider,
		registry:           reg,
		mem:                mem,
		callback:           cb,
		systemPrompt:       defaultSystemPrompt,
		maxIter:            5,
		toolTimeout:        30 * time.Second,
		maxOutputLength:    4096,
		parallelToolCalls:  true,
		maxContextMessages: 50,
	}
	for _, o := range opts {
		o(e)
	}
	return e
}

// AgentConfig holds per-request agent configuration resolved from the Agent model.
type AgentConfig struct {
	Prompt            string
	Tools             []string
	ModelConfig       *models.AgentModelConfig
	MaxIterations     int
	ParallelToolCalls *bool
}

// StreamEvent is emitted during streaming.
type StreamEvent struct {
	Type       string `json:"type"`
	Content    string `json:"content,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	Arguments  string `json:"arguments,omitempty"`
	Result     string `json:"result,omitempty"`
	Success    bool   `json:"success,omitempty"`
	Error      error  `json:"-"`
	Iteration  int    `json:"iteration,omitempty"`
}

// ProcessStream handles a streaming chat turn with the ReAct loop.
func (e *Engine) ProcessStream(ctx context.Context, sessionID, userMessage string, agentCfg *AgentConfig, modelCfg *models.ModelConfig, userID int64) <-chan StreamEvent {
	ch := make(chan StreamEvent, 20)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				ch <- StreamEvent{Type: "error", Error: fmt.Errorf("panic: %v", r)}
			}
		}()

		prompt := e.systemPrompt
		if agentCfg != nil && agentCfg.Prompt != "" {
			prompt = agentCfg.Prompt
		}

		maxIter := e.maxIter
		if agentCfg != nil && agentCfg.MaxIterations > 0 {
			maxIter = agentCfg.MaxIterations
		}

		if err := e.mem.AddMessage(ctx, sessionID, memory.Message{Role: "user", Content: userMessage}, userID); err != nil {
			ch <- StreamEvent{Type: "error", Error: fmt.Errorf("save user message: %w", err)}
			return
		}

		history, _ := e.mem.History(ctx, sessionID)
		messages := e.buildMessages(prompt, history)

		filteredTools := e.resolveTools(agentCfg)
		llmProvider := e.resolveLLMProvider(agentCfg, modelCfg)
		chatOpts := e.buildChatOptions(agentCfg)

		parallelToolCalls := e.parallelToolCalls
		if agentCfg != nil && agentCfg.ParallelToolCalls != nil {
			parallelToolCalls = *agentCfg.ParallelToolCalls
		}

		for iter := 0; iter < maxIter; iter++ {
			ch <- StreamEvent{Type: "thinking", Iteration: iter + 1}
			e.callback.OnLLMStart(ctx, sessionID)

			result, err := llmProvider.Chat(ctx, messages, filteredTools, chatOpts...)
			if err != nil {
				ch <- StreamEvent{Type: "error", Error: err}
				return
			}

			if len(result.ToolCalls) == 0 {
				e.emitContentAsChunks(ch, result.Content)
				e.saveMessage(ctx, sessionID, memory.Message{
					Role: "assistant", Content: result.Content,
				}, userID)
				e.callback.OnLLMEnd(ctx, sessionID, 0, 0)
				ch <- StreamEvent{Type: "done", Content: sessionID}
				return
			}

			toolCallMsg := memory.Message{
				Role:    "assistant_tool_call",
				Content: buildToolCallJSON(result.ToolCalls),
			}
			e.saveMessage(ctx, sessionID, toolCallMsg, userID)
			messages = append(messages, toolCallToLLMMessage(result))

			if parallelToolCalls && len(result.ToolCalls) > 1 {
				toolMsgs := e.executeToolsConcurrently(ctx, ch, result.ToolCalls, sessionID, userID)
				messages = append(messages, toolMsgs...)
			} else {
				for _, tc := range result.ToolCalls {
					toolMsg := e.executeAndEmitTool(ctx, ch, tc, sessionID, userID)
					messages = append(messages, toolMsg)
				}
			}
			e.callback.OnLLMEnd(ctx, sessionID, 0, 0)
		}

		ch <- StreamEvent{Type: "thinking", Content: "max iterations reached, generating final answer"}
		result, err := llmProvider.Chat(ctx, messages, nil, chatOpts...)
		if err != nil {
			ch <- StreamEvent{Type: "error", Error: err}
			return
		}
		e.emitContentAsChunks(ch, result.Content)
		e.saveMessage(ctx, sessionID, memory.Message{
			Role: "assistant", Content: result.Content,
		}, userID)
		e.callback.OnLLMEnd(ctx, sessionID, 0, 0)
		ch <- StreamEvent{Type: "done", Content: sessionID}
	}()

	return ch
}

func (e *Engine) emitContentAsChunks(ch chan<- StreamEvent, content string) {
	if content != "" {
		ch <- StreamEvent{Type: "chunk", Content: content}
	}
}

func (e *Engine) saveMessage(ctx context.Context, sessionID string, msg memory.Message, userID int64) {
	if err := e.mem.AddMessage(ctx, sessionID, msg, userID); err != nil {
		e.callback.Logger.Warn("failed to save message",
			zap.String("session", sessionID),
			zap.String("role", msg.Role),
			zap.Error(err),
		)
	}
}

func (e *Engine) resolveTools(agentCfg *AgentConfig) []tools.Tool {
	if agentCfg == nil || len(agentCfg.Tools) == 0 {
		return nil
	}
	return e.registry.GetByNames(agentCfg.Tools)
}

func (e *Engine) resolveLLMProvider(agentCfg *AgentConfig, userCfg *models.ModelConfig) *llm.Provider {
	model, baseURL, token := "", "", ""

	if userCfg != nil && userCfg.Model != "" {
		model = userCfg.Model
		baseURL = userCfg.BaseURL
		token = userCfg.Token
	}

	if agentCfg != nil && agentCfg.ModelConfig != nil {
		if agentCfg.ModelConfig.Model != "" {
			model = agentCfg.ModelConfig.Model
		}
	}

	if model == "" {
		return e.llm
	}

	if p, err := e.llm.WithOverride(model, baseURL, token); err == nil {
		return p
	}
	return e.llm
}

func (e *Engine) buildChatOptions(agentCfg *AgentConfig) []llm.ChatOption {
	if agentCfg == nil || agentCfg.ModelConfig == nil {
		return nil
	}
	var opts []llm.ChatOption
	if agentCfg.ModelConfig.Temperature != nil {
		opts = append(opts, llm.WithTemperature(*agentCfg.ModelConfig.Temperature))
	}
	if agentCfg.ModelConfig.MaxTokens > 0 {
		opts = append(opts, llm.WithChatMaxTokens(agentCfg.ModelConfig.MaxTokens))
	}
	if agentCfg.ModelConfig.TopP != nil {
		opts = append(opts, llm.WithTopP(*agentCfg.ModelConfig.TopP))
	}
	return opts
}

func (e *Engine) executeToolWithTimeout(ctx context.Context, name, input string) (output string, err error) {
	timeout := e.toolTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	maxOutputLen := e.maxOutputLength
	if maxOutputLen == 0 {
		maxOutputLen = 4096
	}

	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool %q panicked: %v", name, r)
		}
	}()

	traceID := tracing.FromContext(ctx).TraceID
	span := e.callback.OnToolStart(ctx, traceID, name, input)
	output, err = e.registry.Call(toolCtx, name, input)
	e.callback.OnToolEnd(span, name, output)

	if len(output) > maxOutputLen {
		output = output[:maxOutputLen] + "\n...(output truncated at " + strconv.Itoa(maxOutputLen) + " chars, data may be incomplete)"
	}

	return
}

func (e *Engine) executeAndEmitTool(ctx context.Context, ch chan<- StreamEvent, tc llms.ToolCall, sessionID string, userID int64) llm.MessageContent {
	name, args := "", ""
	if tc.FunctionCall != nil {
		name = tc.FunctionCall.Name
		args = tc.FunctionCall.Arguments
	}

	ch <- StreamEvent{
		Type: "tool_start", ToolCallID: tc.ID,
		ToolName: name, Arguments: args,
	}

	output, err := e.executeToolWithTimeout(ctx, name, args)
	success := err == nil
	resultStr := output
	if !success {
		resultStr = fmt.Sprintf("Error: %v", err)
	}

	ch <- StreamEvent{
		Type: "tool_end", ToolCallID: tc.ID,
		ToolName: name, Result: resultStr, Success: success,
	}

	e.saveMessage(ctx, sessionID, memory.Message{
		Role:    "tool",
		Content: buildToolResultJSON(tc.ID, name, resultStr),
	}, userID)

	return llm.MessageContent{
		Role: llm.RoleTool, Content: resultStr,
		ToolCallID: tc.ID, ToolCallName: name,
	}
}

func (e *Engine) executeToolsConcurrently(
	ctx context.Context, ch chan<- StreamEvent,
	toolCalls []llms.ToolCall, sessionID string, userID int64,
) []llm.MessageContent {
	type toolResult struct {
		Index   int
		CallID  string
		Name    string
		Output  string
		Success bool
		Message llm.MessageContent
	}

	results := make([]toolResult, len(toolCalls))
	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxParallelTools)

	for i, tc := range toolCalls {
		i, tc := i, tc
		name, args := "", ""
		if tc.FunctionCall != nil {
			name = tc.FunctionCall.Name
			args = tc.FunctionCall.Arguments
		}

		ch <- StreamEvent{
			Type: "tool_start", ToolCallID: tc.ID,
			ToolName: name, Arguments: args,
		}

		g.Go(func() error {
			output, err := e.executeToolWithTimeout(gCtx, name, args)
			success := err == nil
			resultStr := output
			if !success {
				resultStr = fmt.Sprintf("Error: %v", err)
			}

			mu.Lock()
			results[i] = toolResult{
				Index: i, CallID: tc.ID, Name: name,
				Output: resultStr, Success: success,
				Message: llm.MessageContent{
					Role: llm.RoleTool, Content: resultStr,
					ToolCallID: tc.ID, ToolCallName: name,
				},
			}
			mu.Unlock()
			return nil
		})
	}
	g.Wait()

	var msgs []llm.MessageContent
	for _, r := range results {
		ch <- StreamEvent{
			Type: "tool_end", ToolCallID: r.CallID,
			ToolName: r.Name, Result: r.Output, Success: r.Success,
		}
		msgs = append(msgs, r.Message)

		e.saveMessage(ctx, sessionID, memory.Message{
			Role:    "tool",
			Content: buildToolResultJSON(r.CallID, r.Name, r.Output),
		}, userID)
	}
	return msgs
}

func toolCallToLLMMessage(result *llm.ChatResult) llm.MessageContent {
	return llm.MessageContent{
		Role:      llm.RoleAssistant,
		Content:   result.Content,
		ToolCalls: result.ToolCalls,
	}
}

func (e *Engine) buildMessages(systemPrompt string, history []memory.Message) []llm.MessageContent {
	trimmed := trimHistory(history, e.maxContextMessages)

	msgs := []llm.MessageContent{{Role: llm.RoleSystem, Content: systemPrompt}}
	for _, m := range trimmed {
		switch m.Role {
		case "user":
			msgs = append(msgs, llm.MessageContent{Role: llm.RoleUser, Content: m.Content})
		case "assistant":
			msgs = append(msgs, llm.MessageContent{Role: llm.RoleAssistant, Content: m.Content})
		case "assistant_tool_call":
			var tc toolCallContent
			if err := json.Unmarshal([]byte(m.Content), &tc); err == nil {
				msg := llm.MessageContent{Role: llm.RoleAssistant}
				for _, call := range tc.ToolCalls {
					msg.ToolCalls = append(msg.ToolCalls, llms.ToolCall{
						ID:   call.ID,
						Type: "function",
						FunctionCall: &llms.FunctionCall{
							Name: call.Name, Arguments: call.Arguments,
						},
					})
				}
				msgs = append(msgs, msg)
			}
		case "tool":
			var tr toolResultContent
			if err := json.Unmarshal([]byte(m.Content), &tr); err == nil {
				msgs = append(msgs, llm.MessageContent{
					Role: llm.RoleTool, Content: tr.Result,
					ToolCallID: tr.ToolCallID, ToolCallName: tr.Name,
				})
			}
		}
	}
	return msgs
}

// trimHistory keeps the most recent messages while preserving tool_call/tool pairs.
func trimHistory(history []memory.Message, maxMessages int) []memory.Message {
	if maxMessages <= 0 || len(history) <= maxMessages {
		return history
	}

	start := len(history) - maxMessages

	// Ensure we don't split a tool_call/tool pair — if the message at start
	// is a "tool" response, back up to include its preceding assistant_tool_call.
	for start > 0 && history[start].Role == "tool" {
		start--
	}

	return history[start:]
}
