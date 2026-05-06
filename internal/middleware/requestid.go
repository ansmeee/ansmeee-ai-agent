package middleware

import (
	"ansmeee-ai-agent/internal/tracing"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	HeaderRequestID = "X-Request-ID"
	HeaderTraceID   = "X-Trace-ID"
	CtxRequestID    = "request_id"
)

// RequestID injects or forwards a request ID and trace context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(HeaderRequestID)
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Set(CtxRequestID, rid)
		c.Header(HeaderRequestID, rid)

		// Inject trace context: request ID doubles as trace ID.
		t := tracing.Trace{
			TraceID: c.GetHeader(HeaderTraceID),
			SpanID:  tracing.GenID(8),
		}
		if t.TraceID == "" {
			t.TraceID = rid
		}
		c.Header(HeaderTraceID, t.TraceID)
		ctx := tracing.ContextWithTrace(c.Request.Context(), t)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
