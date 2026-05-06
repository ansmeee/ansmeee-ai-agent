package handler

import (
	"fmt"
	"net/http"
	"strings"

	"ansmeee-ai-agent/internal/agent"
	"ansmeee-ai-agent/internal/llm"
	"ansmeee-ai-agent/internal/memory"
	"ansmeee-ai-agent/internal/middleware"
	"ansmeee-ai-agent/internal/models"
	"ansmeee-ai-agent/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamHandler handles SSE streaming chat requests.
type StreamHandler struct {
	engine           *agent.Engine
	mem              memory.SessionStore
	agentStore       *agent.AgentStore
	modelConfigStore *llm.ModelConfigStore
}

// NewStreamHandler creates a new stream handler.
func NewStreamHandler(engine *agent.Engine, mem memory.SessionStore, agentStore *agent.AgentStore, modelConfigStore *llm.ModelConfigStore) *StreamHandler {
	return &StreamHandler{engine: engine, mem: mem, agentStore: agentStore, modelConfigStore: modelConfigStore}
}

func (h *StreamHandler) resolvePrompt(agentID string) string {
	if agentID != "" && h.agentStore != nil {
		if a, err := h.agentStore.Get(agentID); err == nil {
			return a.Prompt
		}
	}
	return ""
}

// Handle processes a streaming chat request via SSE.
func (h *StreamHandler) Handle(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "message is required")
		return
	}
	if req.Message == "" {
		response.BadRequest(c, "message is required")
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Set SSE headers.
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.InternalError(c, "streaming not supported")
		return
	}

	// Record agent for this session.
	userID := c.GetInt64(middleware.CtxUserID)
	if req.AgentID != "" {
		_ = h.mem.SetAgent(c.Request.Context(), sessionID, req.AgentID, userID)
	}

	// Send session_id as first event.
	writeSSE(c.Writer, flusher, "session", sessionID)

	// Process stream.
	done := make(chan struct{})
	defer func() {
		select {
		case <-done:
		default:
			writeSSE(c.Writer, flusher, "error", "stream ended unexpectedly")
			flusher.Flush()
		}
	}()

	var modelCfg *models.ModelConfig
	if h.modelConfigStore != nil {
		modelCfg, _ = h.modelConfigStore.GetByUser(userID)
	}
	ch := h.engine.ProcessStream(c.Request.Context(), sessionID, req.Message, h.resolvePrompt(req.AgentID), modelCfg, userID)

	for evt := range ch {
		switch evt.Type {
		case "chunk":
			writeSSE(c.Writer, flusher, "chunk", evt.Content)
		case "done":
			writeSSE(c.Writer, flusher, "done", fmt.Sprintf(`{"session_id":"%s"}`, evt.Content))
			flusher.Flush()
			close(done)
			return
		case "error":
			writeSSE(c.Writer, flusher, "error", evt.Error.Error())
			flusher.Flush()
			close(done)
			return
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
	flusher.Flush()
}
