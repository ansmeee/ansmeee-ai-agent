package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"time"

	"go.uber.org/zap"
)

type contextKey string

const (
	keyTraceID    contextKey = "trace_id"
	keySpanID     contextKey = "span_id"
	headerTraceID            = "X-Trace-ID"
)

// Trace holds trace identifiers.
type Trace struct {
	TraceID string
	SpanID  string
}

// NewTrace generates a new trace.
func NewTrace() Trace {
	return Trace{
		TraceID: genID(16),
		SpanID:  genID(8),
	}
}

// NewSpan creates a child span from the current trace context.
func NewSpan(ctx context.Context, name string) (context.Context, Span) {
	parent := FromContext(ctx)
	if parent.TraceID == "" {
		parent = NewTrace()
	}
	span := Span{
		Trace: Trace{
			TraceID: parent.TraceID,
			SpanID:  genID(8),
		},
		Name:      name,
		StartTime: time.Now(),
		parent:    &parent,
	}
	return ContextWithTrace(ctx, span.Trace), span
}

// Span represents a single operation within a trace.
type Span struct {
	Trace
	Name      string
	StartTime time.Time
	parent    *Trace
}

// End marks the span as complete and optionally logs its duration.
func (s Span) End(logger *zap.Logger, fields ...zap.Field) {
	if logger == nil {
		return
	}
	dur := time.Since(s.StartTime)
	fields = append(fields,
		zap.String("trace_id", s.TraceID),
		zap.String("span_id", s.SpanID),
		zap.String("span", s.Name),
		zap.Duration("duration_ms", dur),
	)
	logger.Debug("span", fields...)
}

// ContextWithTrace injects trace into context.
func ContextWithTrace(ctx context.Context, t Trace) context.Context {
	ctx = context.WithValue(ctx, keyTraceID, t.TraceID)
	ctx = context.WithValue(ctx, keySpanID, t.SpanID)
	return ctx
}

// FromContext extracts trace info from context.
func FromContext(ctx context.Context) Trace {
	traceID, _ := ctx.Value(keyTraceID).(string)
	spanID, _ := ctx.Value(keySpanID).(string)
	return Trace{TraceID: traceID, SpanID: spanID}
}

// FromHeader extracts trace from HTTP header or generates a new one.
func FromHeader(r io.Reader) Trace {
	// If we had http.Header, we'd read the header. For now, generate new.
	return NewTrace()
}

// ZapFields returns zap fields for the current trace context.
func ZapFields(ctx context.Context) []zap.Field {
	t := FromContext(ctx)
	if t.TraceID == "" {
		return nil
	}
	return []zap.Field{
		zap.String("trace_id", t.TraceID),
		zap.String("span_id", t.SpanID),
	}
}

// GenID generates a random hex ID of n bytes.
func GenID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func genID(n int) string {
	return GenID(n)
}
