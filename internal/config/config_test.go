package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_AgentConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	// Minimal config without agent section — defaults should apply
	if err := os.WriteFile(cfgPath, []byte("server:\n  port: 9999\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Agent.MaxIterations != 5 {
		t.Errorf("MaxIterations = %d, want 5", cfg.Agent.MaxIterations)
	}
	if cfg.Agent.ToolTimeout != 30*time.Second {
		t.Errorf("ToolTimeout = %v, want 30s", cfg.Agent.ToolTimeout)
	}
	if cfg.Agent.MaxOutputLength != 4096 {
		t.Errorf("MaxOutputLength = %d, want 4096", cfg.Agent.MaxOutputLength)
	}
	if !cfg.Agent.ParallelToolCalls {
		t.Error("ParallelToolCalls should default to true")
	}
	if cfg.Agent.MaxContextMessages != 50 {
		t.Errorf("MaxContextMessages = %d, want 50", cfg.Agent.MaxContextMessages)
	}
}

func TestLoad_AgentConfigExplicit(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `
server:
  port: 8080
agent:
  max_iterations: 10
  tool_timeout: 60s
  max_output_length: 8192
  parallel_tool_calls: false
  max_context_messages: 100
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Agent.MaxIterations != 10 {
		t.Errorf("MaxIterations = %d, want 10", cfg.Agent.MaxIterations)
	}
	if cfg.Agent.ToolTimeout != 60*time.Second {
		t.Errorf("ToolTimeout = %v, want 60s", cfg.Agent.ToolTimeout)
	}
	if cfg.Agent.MaxOutputLength != 8192 {
		t.Errorf("MaxOutputLength = %d, want 8192", cfg.Agent.MaxOutputLength)
	}
	if cfg.Agent.ParallelToolCalls {
		t.Error("ParallelToolCalls should be false when explicitly set")
	}
	if cfg.Agent.MaxContextMessages != 100 {
		t.Errorf("MaxContextMessages = %d, want 100", cfg.Agent.MaxContextMessages)
	}
}
