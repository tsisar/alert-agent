package anthropic

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/tsisar/alert-agent/internal/llm"
)

func TestConvertMessages_CoalescesToolResults(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "investigate"},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{
			{ID: "call_1", Name: "query", Arguments: `{"q":"up"}`},
			{ID: "call_2", Name: "panel", Arguments: `{}`},
		}},
		{Role: llm.RoleTool, ToolCallID: "call_1", Content: "result one"},
		{Role: llm.RoleTool, ToolCallID: "call_2", Content: ""},
		{Role: llm.RoleAssistant, Content: "done"},
	}

	out, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages: %v", err)
	}

	// user, assistant(tool_use), user(2 tool_results), assistant(text)
	if len(out) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(out))
	}

	if out[0].Role != anthropic.MessageParamRoleUser {
		t.Errorf("msg[0] role = %s, want user", out[0].Role)
	}

	if out[1].Role != anthropic.MessageParamRoleAssistant {
		t.Errorf("msg[1] role = %s, want assistant", out[1].Role)
	}
	if len(out[1].Content) != 2 {
		t.Errorf("assistant tool_use blocks = %d, want 2", len(out[1].Content))
	}

	// The two consecutive tool results must collapse into one user message.
	if out[2].Role != anthropic.MessageParamRoleUser {
		t.Errorf("msg[2] role = %s, want user", out[2].Role)
	}
	if len(out[2].Content) != 2 {
		t.Fatalf("coalesced tool_result blocks = %d, want 2", len(out[2].Content))
	}

	// Empty tool output is replaced with a placeholder so the API accepts it.
	second := out[2].Content[1].OfToolResult
	if second == nil {
		t.Fatal("second block is not a tool_result")
	}
	if len(second.Content) == 0 || second.Content[0].OfText == nil || second.Content[0].OfText.Text == "" {
		t.Error("empty tool result content was not replaced with a placeholder")
	}

	if out[3].Role != anthropic.MessageParamRoleAssistant {
		t.Errorf("msg[3] role = %s, want assistant", out[3].Role)
	}
}
