package agent

import (
	"encoding/json"

	"github.com/tmc/langchaingo/llms"
)

type toolCallEntry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCallContent struct {
	ToolCalls []toolCallEntry `json:"tool_calls"`
}

type toolResultContent struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Result     string `json:"result"`
}

// buildToolCallJSON serializes assistant tool calls into a JSON string
// for storage as Message.Content with role=assistant_tool_call.
func buildToolCallJSON(toolCalls []llms.ToolCall) string {
	entries := make([]toolCallEntry, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name, args := "", ""
		if tc.FunctionCall != nil {
			name = tc.FunctionCall.Name
			args = tc.FunctionCall.Arguments
		}
		entries = append(entries, toolCallEntry{
			ID:        tc.ID,
			Name:      name,
			Arguments: args,
		})
	}
	b, _ := json.Marshal(toolCallContent{ToolCalls: entries})
	return string(b)
}

// buildToolResultJSON serializes a tool execution result into a JSON string
// for storage as Message.Content with role=tool.
func buildToolResultJSON(callID, name, result string) string {
	b, _ := json.Marshal(toolResultContent{
		ToolCallID: callID,
		Name:       name,
		Result:     result,
	})
	return string(b)
}
