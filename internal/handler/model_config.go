package handler

import (
	"ansmeee-ai-agent/internal/llm"
	"ansmeee-ai-agent/internal/middleware"
	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
)

// ModelConfigHandler handles user model config API.
type ModelConfigHandler struct {
	store *llm.ModelConfigStore
}

// NewModelConfigHandler creates a new model config handler.
func NewModelConfigHandler(store *llm.ModelConfigStore) *ModelConfigHandler {
	return &ModelConfigHandler{store: store}
}

// Get returns the model config for the current user.
func (h *ModelConfigHandler) Get(c *gin.Context) {
	cfg, err := h.store.GetByUser(c.GetInt64(middleware.CtxUserID)) // default user_id=1
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, cfg)
}

type modelConfigRequest struct {
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

// Save upserts the model config for the current user.
func (h *ModelConfigHandler) Save(c *gin.Context) {
	var req modelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if req.Model == "" || req.Token == "" {
		response.BadRequest(c, "model and token are required")
		return
	}
	cfg, err := h.store.Save(1, req.Model, req.BaseURL, req.Token)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, cfg)
}
