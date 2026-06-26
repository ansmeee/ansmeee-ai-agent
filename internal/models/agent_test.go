package models

import (
	"encoding/json"
	"testing"
)

func TestAgentModelConfig_TemperatureZero(t *testing.T) {
	temp := 0.0
	cfg := AgentModelConfig{
		Model:       "test-model",
		Temperature: &temp,
		MaxTokens:   100,
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	s := string(b)
	if !jsonContains(s, `"temperature":0`) {
		t.Errorf("temperature=0 should be present in JSON, got: %s", s)
	}
}

func TestAgentModelConfig_TemperatureNil(t *testing.T) {
	cfg := AgentModelConfig{
		Model:     "test-model",
		MaxTokens: 100,
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	s := string(b)
	if jsonContains(s, `"temperature"`) {
		t.Errorf("nil temperature should be omitted, got: %s", s)
	}
}

func TestAgentModelConfig_TopPZero(t *testing.T) {
	topP := 0.0
	cfg := AgentModelConfig{TopP: &topP}

	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	s := string(b)
	if !jsonContains(s, `"top_p":0`) {
		t.Errorf("top_p=0 should be present in JSON, got: %s", s)
	}
}

func TestAgentModelConfig_RoundTrip(t *testing.T) {
	temp := 0.7
	topP := 0.9
	original := AgentModelConfig{
		Model:       "gpt-4",
		Temperature: &temp,
		MaxTokens:   2048,
		TopP:        &topP,
	}

	b, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded AgentModelConfig
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Model != original.Model {
		t.Errorf("Model = %q, want %q", decoded.Model, original.Model)
	}
	if decoded.Temperature == nil || *decoded.Temperature != *original.Temperature {
		t.Errorf("Temperature = %v, want %v", decoded.Temperature, *original.Temperature)
	}
	if decoded.MaxTokens != original.MaxTokens {
		t.Errorf("MaxTokens = %d, want %d", decoded.MaxTokens, original.MaxTokens)
	}
	if decoded.TopP == nil || *decoded.TopP != *original.TopP {
		t.Errorf("TopP = %v, want %v", decoded.TopP, *original.TopP)
	}
}

func jsonContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
