package handler

import (
	"net/http"

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

// Schema returns the parameter schema for a specific tool.
func (h *ToolHandler) Schema(c *gin.Context) {
	name := c.Param("name")
	schema := h.registry.GetSchema(name)
	if schema == nil {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "tool not found")
		return
	}
	response.OK(c, schema)
}
