package tool

import (
	"context"
	"fmt"
	"sync"

	"github.com/tmc/langchaingo/tools"
)

// Registry manages available tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]tools.Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tools.Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t tools.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := t.Name()
	if _, ok := r.tools[name]; ok {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = t
	return nil
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (tools.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tool descriptions.
func (r *Registry) List() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infos := make([]ToolInfo, 0, len(r.tools))
	for _, t := range r.tools {
		infos = append(infos, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
		})
	}
	return infos
}

// All returns all registered tools as a slice.
func (r *Registry) All() []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]tools.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Call executes a tool by name.
func (r *Registry) Call(ctx context.Context, name, input string) (string, error) {
	t, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("tool %q not found", name)
	}
	return t.Call(ctx, input)
}

// ToolInfo is a lightweight tool descriptor for API responses.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
