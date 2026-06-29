package handler

import (
	"net/http"

	"ansmeee-ai-agent/internal/agent"
	"ansmeee-ai-agent/internal/middleware"
	"ansmeee-ai-agent/internal/models"
	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
)

// AgentHandler handles agent CRUD requests.
type AgentHandler struct {
	store *agent.AgentStore
}

// NewAgentHandler creates a new agent handler.
func NewAgentHandler(store *agent.AgentStore) *AgentHandler {
	return &AgentHandler{store: store}
}

type agentRequest struct {
	Title         string                   `json:"title"`
	Description   string                   `json:"description"`
	Prompt        string                   `json:"prompt"`
	Tools         []string                 `json:"tools"`
	ModelConfig   *models.AgentModelConfig `json:"model_config"`
	MaxIterations int8                     `json:"max_iterations"`
	Status        *int8                    `json:"status"`
}

func (h *AgentHandler) userID(c *gin.Context) int64 {
	return c.GetInt64(middleware.CtxUserID)
}

// List returns agents for the current user.
func (h *AgentHandler) List(c *gin.Context) {
	uid := h.userID(c)
	_ = h.store.EnsureDefault(uid)
	agents := h.store.List(uid)
	if agents == nil {
		agents = []*models.Agent{}
	}
	response.OK(c, gin.H{"agents": agents})
}

// Get returns a single agent.
func (h *AgentHandler) Get(c *gin.Context) {
	a, err := h.store.Get(c.Param("id"), h.userID(c))
	if err != nil {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, err.Error())
		return
	}
	response.OK(c, a)
}

// Create adds a new agent.
func (h *AgentHandler) Create(c *gin.Context) {
	var req agentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if req.Title == "" {
		response.BadRequest(c, "title is required")
		return
	}
	if req.Prompt == "" {
		response.BadRequest(c, "prompt is required")
		return
	}
	a, err := h.store.Create(h.userID(c), req.Title, req.Description, req.Prompt,
		req.Tools, req.ModelConfig, req.MaxIterations)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, a)
}

// Update modifies an existing agent.
func (h *AgentHandler) Update(c *gin.Context) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	a, err := h.store.Update(c.Param("id"), h.userID(c), raw)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, a)
}

// Delete removes an agent.
func (h *AgentHandler) Delete(c *gin.Context) {
	if err := h.store.Delete(c.Param("id"), h.userID(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
