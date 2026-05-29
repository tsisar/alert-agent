package anthropic

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/tsisar/alert-agent/internal/llm"
)

func TestCacheLastMessage(t *testing.T) {
	msgs := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("first turn")),
		anthropic.NewUserMessage(
			anthropic.NewToolResultBlock("call_1", "result a", false),
			anthropic.NewToolResultBlock("call_2", "result b", false),
		),
	}

	cacheLastMessage(msgs)

	last, _ := json.Marshal(msgs[len(msgs)-1])
	if !strings.Contains(string(last), "cache_control") {
		t.Errorf("expected cache_control on the last message, got %s", last)
	}

	first, _ := json.Marshal(msgs[0])
	if strings.Contains(string(first), "cache_control") {
		t.Errorf("earlier messages should not carry a cache breakpoint, got %s", first)
	}
}

func TestConvertMessages_ThinkingBlocksComeFirst(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "investigate"},
		{
			Role: llm.RoleAssistant,
			Thinking: []llm.ThinkingBlock{
				{Text: "let me check the dashboard", Signature: "sig-abc"},
			},
			ToolCalls: []llm.ToolCall{{ID: "call_1", Name: "query", Arguments: `{}`}},
		},
	}

	out, err := convertMessages(messages)
	if err != nil {
		t.Fatalf("convertMessages: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}

	assistant := out[1]
	if len(assistant.Content) != 2 {
		t.Fatalf("assistant blocks = %d, want 2 (thinking + tool_use)", len(assistant.Content))
	}

	// The thinking block must be first, with its signature preserved.
	thinking := assistant.Content[0].OfThinking
	if thinking == nil {
		t.Fatal("first assistant block is not a thinking block")
	}
	if thinking.Thinking != "let me check the dashboard" || thinking.Signature != "sig-abc" {
		t.Errorf("thinking block not preserved: %+v", thinking)
	}

	if assistant.Content[1].OfToolUse == nil {
		t.Error("second assistant block is not a tool_use block")
	}
}

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
