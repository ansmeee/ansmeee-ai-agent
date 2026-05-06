package handler

import (
	"ansmeee-ai-agent/internal/tool"
	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
)

// ToolHandler handles tool listing requests.
type ToolHandler struct {
	registry *tool.Registry
}

// NewToolHandler creates a new tool handler.
func NewToolHandler(registry *tool.Registry) *ToolHandler {
	return &ToolHandler{registry: registry}
}

// Handle returns all available tools.
func (h *ToolHandler) Handle(c *gin.Context) {
	tools := h.registry.List()
	if tools == nil {
		tools = []tool.ToolInfo{}
	}
	response.OK(c, gin.H{"tools": tools})
}
