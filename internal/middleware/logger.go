package middleware

import (
	"time"

	"ansmeee-ai-agent/internal/tracing"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger logs each request using zap with trace context.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		t := tracing.FromContext(c.Request.Context())

		logger.Info("request",
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("request_id", c.GetString(CtxRequestID)),
			zap.String("trace_id", t.TraceID),
			zap.String("span_id", t.SpanID),
		)
	}
}
