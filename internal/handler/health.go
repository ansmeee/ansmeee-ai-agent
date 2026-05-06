package handler

import (
	"time"

	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
)

// HealthCheck returns service status.
func HealthCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
