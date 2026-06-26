package agent

import (
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func TestBuildToolCallJSON(t *testing.T) {
	calls := []llms.ToolCall{
		{
			ID:   "call_1",
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      "calculator",
				Arguments: `{"expr":"2+2"}`,
			},
		},
		{
			ID:   "call_2",
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      "datetime",
				Arguments: `{"format":"now"}`,
			},
		},
	}

	result := buildToolCallJSON(calls)

	if result == "" {
		t.Fatal("expected non-empty JSON")
	}
	if !contains(result, `"tool_calls"`) {
		t.Errorf("expected tool_calls key, got: %s", result)
	}
	if !contains(result, `"call_1"`) || !contains(result, `"call_2"`) {
		t.Errorf("expected both call IDs, got: %s", result)
	}
	if !contains(result, `"calculator"`) || !contains(result, `"datetime"`) {
		t.Errorf("expected tool names, got: %s", result)
	}
}

func TestBuildToolCallJSON_NilFunctionCall(t *testing.T) {
	calls := []llms.ToolCall{
		{ID: "call_1", Type: "function"},
	}
	result := buildToolCallJSON(calls)
	if !contains(result, `"name":""`) {
		t.Errorf("expected empty name for nil FunctionCall, got: %s", result)
	}
}

func TestBuildToolResultJSON(t *testing.T) {
	result := buildToolResultJSON("call_1", "calculator", "4")

	if result == "" {
		t.Fatal("expected non-empty JSON")
	}
	if !contains(result, `"tool_call_id":"call_1"`) {
		t.Errorf("expected tool_call_id, got: %s", result)
	}
	if !contains(result, `"name":"calculator"`) {
		t.Errorf("expected name, got: %s", result)
	}
	if !contains(result, `"result":"4"`) {
		t.Errorf("expected result, got: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
