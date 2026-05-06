package middleware

import (
	"net/http"

	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery catches panics and returns a 500 error.
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("request_id", c.GetString(CtxRequestID)),
				)
				response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
