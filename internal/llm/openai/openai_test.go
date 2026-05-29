package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tsisar/alert-agent/internal/llm"
)

type fakeCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Choices []fakeChoice `json:"choices"`
}

type fakeChoice struct {
	Index        int         `json:"index"`
	Message      fakeMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type fakeMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []fakeToolCall `json:"tool_calls,omitempty"`
}

type fakeToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function fakeFunctionCall `json:"function"`
}

type fakeFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func newFakeOpenAIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return ts
}

func TestChatCompletion_SimpleResponse(t *testing.T) {
	ts := newFakeOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}

		resp := fakeCompletionResponse{
			ID:     "chatcmpl-test",
			Object: "chat.completion",
			Choices: []fakeChoice{
				{
					Index: 0,
					Message: fakeMessage{
						Role:    "assistant",
						Content: "CPU usage is critically high at 95%.",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	provider := New("test-key", ts.URL, "gpt-4o", "")

	resp, err := provider.ChatCompletion(context.Background(), &llm.ChatRequest{
		SystemPrompt: "You are an alert investigator.",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "CPU alert fired on web-1"},
		},
		MaxTokens: 1024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Content != "CPU usage is critically high at 95%." {
		t.Fatalf("unexpected content: %q", resp.Content)
	}
	if resp.FinishReason != llm.FinishReasonStop {
		t.Fatalf("expected finish_reason=stop, got %q", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestChatCompletion_WithToolCalls(t *testing.T) {
	ts := newFakeOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := fakeCompletionResponse{
			ID:     "chatcmpl-tools",
			Object: "chat.completion",
			Choices: []fakeChoice{
				{
					Index: 0,
					Message: fakeMessage{
						Role:    "assistant",
						Content: "",
						ToolCalls: []fakeToolCall{
							{
								ID:   "call_abc123",
								Type: "function",
								Function: fakeFunctionCall{
									Name:      "query_prometheus",
									Arguments: `{"query":"up{job=\"web\"}","time":"2026-03-30T04:00:00Z"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	provider := New("test-key", ts.URL, "gpt-4o", "")

	resp, err := provider.ChatCompletion(context.Background(), &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Check if web service is up"},
		},
		Tools: []llm.Tool{
			{
				Name:        "query_prometheus",
				Description: "Run a PromQL query",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{"type": "string"},
						"time":  map[string]any{"type": "string"},
					},
					"required": []string{"query"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.FinishReason != llm.FinishReasonToolCall {
		t.Fatalf("expected finish_reason=tool_call, got %q", resp.FinishReason)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	tc := resp.ToolCalls[0]
	if tc.ID != "call_abc123" {
		t.Fatalf("expected tool call ID 'call_abc123', got %q", tc.ID)
	}
	if tc.Name != "query_prometheus" {
		t.Fatalf("expected tool name 'query_prometheus', got %q", tc.Name)
	}
}

func TestChatCompletion_ToolResultRoundTrip(t *testing.T) {
	callCount := 0
	ts := newFakeOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var resp fakeCompletionResponse
		if callCount == 1 {
			resp = fakeCompletionResponse{
				ID:     "chatcmpl-1",
				Object: "chat.completion",
				Choices: []fakeChoice{
					{
						Message: fakeMessage{
							Role: "assistant",
							ToolCalls: []fakeToolCall{
								{
									ID:   "call_1",
									Type: "function",
									Function: fakeFunctionCall{
										Name:      "echo",
										Arguments: `{"message":"ping"}`,
									},
								},
							},
						},
						FinishReason: "tool_calls",
					},
				},
			}
		} else {
			resp = fakeCompletionResponse{
				ID:     "chatcmpl-2",
				Object: "chat.completion",
				Choices: []fakeChoice{
					{
						Message: fakeMessage{
							Role:    "assistant",
							Content: "The echo tool returned: pong",
						},
						FinishReason: "stop",
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	provider := New("test-key", ts.URL, "gpt-4o", "")
	ctx := context.Background()

	// First call: model requests tool use
	resp1, err := provider.ChatCompletion(ctx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Echo ping"},
		},
		Tools: []llm.Tool{
			{Name: "echo", Description: "Echo tool", Parameters: map[string]any{"type": "object"}},
		},
	})
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if resp1.FinishReason != llm.FinishReasonToolCall {
		t.Fatalf("expected tool_call, got %q", resp1.FinishReason)
	}

	// Second call: provide tool result
	resp2, err := provider.ChatCompletion(ctx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "Echo ping"},
			{Role: llm.RoleAssistant, ToolCalls: resp1.ToolCalls},
			{Role: llm.RoleTool, Content: "pong", ToolCallID: "call_1"},
		},
		Tools: []llm.Tool{
			{Name: "echo", Description: "Echo tool", Parameters: map[string]any{"type": "object"}},
		},
	})
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if resp2.FinishReason != llm.FinishReasonStop {
		t.Fatalf("expected stop, got %q", resp2.FinishReason)
	}
	if resp2.Content != "The echo tool returned: pong" {
		t.Fatalf("unexpected content: %q", resp2.Content)
	}
}

func TestChatCompletion_EmptyChoices(t *testing.T) {
	ts := newFakeOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := fakeCompletionResponse{
			ID:      "chatcmpl-empty",
			Object:  "chat.completion",
			Choices: []fakeChoice{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	provider := New("test-key", ts.URL, "gpt-4o", "")

	_, err := provider.ChatCompletion(context.Background(), &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestChatCompletion_ServerError(t *testing.T) {
	ts := newFakeOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"internal error","type":"server_error"}}`))
	})

	provider := New("test-key", ts.URL, "gpt-4o", "")

	_, err := provider.ChatCompletion(context.Background(), &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "hello"},
		},
	})
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestChatCompletion_ModelOverride(t *testing.T) {
	var receivedModel string

	ts := newFakeOpenAIServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedModel, _ = body["model"].(string)

		resp := fakeCompletionResponse{
			ID:     "chatcmpl-model",
			Object: "chat.completion",
			Choices: []fakeChoice{
				{
					Message:      fakeMessage{Role: "assistant", Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	provider := New("test-key", ts.URL, "gpt-4o", "")

	_, err := provider.ChatCompletion(context.Background(), &llm.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedModel != "gpt-4o-mini" {
		t.Fatalf("expected model 'gpt-4o-mini', got %q", receivedModel)
	}
}

func TestConvertMessages(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
		{Role: llm.RoleAssistant, Content: "hi there"},
		{Role: llm.RoleTool, Content: "result", ToolCallID: "call_1"},
	}

	result, err := convertMessages("system prompt", messages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// system + 3 messages
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}
}

func TestConvertMessages_UnknownRole(t *testing.T) {
	messages := []llm.Message{
		{Role: "alien", Content: "beep"},
	}

	_, err := convertMessages("", messages)
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected llm.FinishReason
	}{
		{"stop", llm.FinishReasonStop},
		{"tool_calls", llm.FinishReasonToolCall},
		{"length", llm.FinishReasonStop},
		{"", llm.FinishReasonStop},
	}

	for _, tt := range tests {
		got := mapFinishReason(tt.input)
		if got != tt.expected {
			t.Errorf("mapFinishReason(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
