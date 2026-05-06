package agent

import (
	"context"

	"ansmeee-ai-agent/internal/tracing"
	"go.uber.org/zap"
)

// Callback receives events during agent processing.
type Callback struct {
	Logger *zap.Logger
}

// NewCallback creates a new callback handler.
func NewCallback(logger *zap.Logger) *Callback {
	return &Callback{Logger: logger}
}

// OnLLMStart is called before the LLM is invoked.
func (c *Callback) OnLLMStart(ctx context.Context, sessionID string) {
	sid := sessionID
	if len(sid) > 8 {
		sid = sid[:8]
	}
	c.Logger.Debug("llm call started",
		zap.String("session", sid),
	)
}

// OnLLMEnd is called after the LLM responds.
func (c *Callback) OnLLMEnd(ctx context.Context, sessionID string, tokensUsed int, durationMs int64) {
	sid := sessionID
	if len(sid) > 8 {
		sid = sid[:8]
	}
	c.Logger.Debug("llm call ended",
		zap.String("session", sid),
		zap.Int("tokens", tokensUsed),
		zap.Int64("duration_ms", durationMs),
	)
}

// OnToolStart is called before a tool is executed.
func (c *Callback) OnToolStart(ctx context.Context, sessionID, toolName, input string) tracing.Span {
	_, span := tracing.NewSpan(ctx, "tool."+toolName)
	c.Logger.Debug("tool started",
		zap.String("tool", toolName),
		zap.String("input", input),
		zap.String("trace_id", span.TraceID),
		zap.String("span_id", span.SpanID),
	)
	return span
}

// OnToolEnd is called after a tool returns.
func (c *Callback) OnToolEnd(span tracing.Span, toolName, output string) {
	c.Logger.Debug("tool ended",
		zap.String("tool", toolName),
		zap.String("output", output),
	)
	span.End(c.Logger,
		zap.String("tool", toolName),
	)
}
