package memory

import (
	"testing"

	"ansmeee-ai-agent/internal/models"
)

func TestRoleStringToInt(t *testing.T) {
	tests := []struct {
		input string
		want  int8
	}{
		{"user", models.RoleUser},
		{"assistant", models.RoleAssistant},
		{"tool", models.RoleTool},
		{"assistant_tool_call", models.RoleAssistantToolCall},
		{"unknown", models.RoleUser},
		{"", models.RoleUser},
	}
	for _, tt := range tests {
		got := roleStringToInt(tt.input)
		if got != tt.want {
			t.Errorf("roleStringToInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestRoleIntToString(t *testing.T) {
	tests := []struct {
		input int8
		want  string
	}{
		{models.RoleUser, "user"},
		{models.RoleAssistant, "assistant"},
		{models.RoleTool, "tool"},
		{models.RoleAssistantToolCall, "assistant_tool_call"},
		{99, "user"},
	}
	for _, tt := range tests {
		got := roleIntToString(tt.input)
		if got != tt.want {
			t.Errorf("roleIntToString(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRoleRoundTrip(t *testing.T) {
	roles := []string{"user", "assistant", "tool", "assistant_tool_call"}
	for _, role := range roles {
		converted := roleIntToString(roleStringToInt(role))
		if converted != role {
			t.Errorf("round-trip failed for %q: got %q", role, converted)
		}
	}
}
