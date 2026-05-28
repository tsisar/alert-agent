package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tsisar/alert-agent/internal/config"
	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/alert-agent/internal/model"
	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/storage"
)

type mockProvider struct {
	responses []*llm.ChatResponse
	calls     int
	requests  []*llm.ChatRequest
	afterCall func(calls int) // optional hook invoked after each response is selected
}

func (m *mockProvider) ChatCompletion(_ context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	m.requests = append(m.requests, req)
	if m.calls >= len(m.responses) {
		return nil, fmt.Errorf("unexpected call %d", m.calls)
	}
	resp := m.responses[m.calls]
	m.calls++
	if m.afterCall != nil {
		m.afterCall(m.calls)
	}
	return resp, nil
}

func testPayload() *model.WebhookPayload {
	return &model.WebhookPayload{
		Receiver: "test",
		Status:   "firing",
		Alerts: []model.Alert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "HighCPU"},
				Annotations: map[string]string{"summary": "CPU is high"},
				Fingerprint: "abc123",
			},
		},
		CommonLabels: map[string]string{"alertname": "HighCPU"},
		GroupKey:     "test-group",
		Title:        "[FIRING:1] HighCPU",
		State:        "alerting",
	}
}

func testScenario() *scenario.Scenario {
	return &scenario.Scenario{
		Name:     "test",
		Prompt:   "Investigate CPU usage.",
		Timeout:  scenario.Duration{Duration: 2 * time.Minute},
		Priority: "high",
		Channels: scenario.Channels{Telegram: "-100123"},
	}
}

func testLLMConfig() config.LLMConfig {
	return config.LLMConfig{MaxTokens: 1024}
}

type mockPromptRepo struct {
	prompts *storage.PromptSet
}

func (m *mockPromptRepo) Get(key string) (string, error) {
	switch key {
	case "system":
		return m.prompts.System, nil
	case "summary":
		return m.prompts.Summary, nil
	case "resolved":
		return m.prompts.Resolved, nil
	case "paused":
		return m.prompts.Paused, nil
	}
	return "", fmt.Errorf("unknown key: %s", key)
}

func (m *mockPromptRepo) GetAll() (*storage.PromptSet, error) {
	cp := *m.prompts
	return &cp, nil
}

func (m *mockPromptRepo) Set(string, string) error { return nil }
func (m *mockPromptRepo) Invalidate()              {}

func testPrompts() storage.PromptRepository {
	return &mockPromptRepo{
		prompts: &storage.PromptSet{
			System:   "You are a test agent.",
			Summary:  "Summarize:\n",
			Resolved: "RESOLVED test.\n\nAlert data:\n",
			Paused:   "PAUSED test.\n\nAlert data:\n",
		},
	}
}

func TestRun_SimpleCompletion(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{Content: "CPU is fine. Severity: OK", FinishReason: llm.FinishReasonStop},
		},
	}
	executor := &mockExecutor{tools: testTools}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(context.Background(), testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Report != "CPU is fine. Severity: OK" {
		t.Fatalf("unexpected report: %q", result.Report)
	}
	if provider.calls != 1 {
		t.Fatalf("expected 1 LLM call, got %d", provider.calls)
	}
	if provider.requests[0].SystemPrompt != "You are a test agent." {
		t.Fatal("system prompt not passed to provider")
	}
}

func TestRun_ToolUseLoop(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{
				FinishReason: llm.FinishReasonToolCall,
				ToolCalls: []llm.ToolCall{
					{ID: "call_1", Name: "query_prometheus", Arguments: `{"query":"up"}`},
				},
			},
			{
				Content:      "Service is up. Severity: OK",
				FinishReason: llm.FinishReasonStop,
			},
		},
	}
	executor := &mockExecutor{tools: testTools}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(context.Background(), testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Report != "Service is up. Severity: OK" {
		t.Fatalf("unexpected report: %q", result.Report)
	}
	if provider.calls != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", provider.calls)
	}

	secondReq := provider.requests[1]
	foundToolMsg := false
	for _, msg := range secondReq.Messages {
		if msg.Role == llm.RoleTool && msg.ToolCallID == "call_1" {
			foundToolMsg = true
			if msg.Content != "called:query_prometheus" {
				t.Fatalf("unexpected tool result: %q", msg.Content)
			}
		}
	}
	if !foundToolMsg {
		t.Fatal("tool result message not found in second request")
	}
}

func TestRun_MultipleToolCalls(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{
				FinishReason: llm.FinishReasonToolCall,
				ToolCalls: []llm.ToolCall{
					{ID: "call_1", Name: "query_prometheus", Arguments: `{}`},
					{ID: "call_2", Name: "get_panel_image", Arguments: `{}`},
				},
			},
			{
				Content:      "Report with evidence. Severity: WARNING",
				FinishReason: llm.FinishReasonStop,
			},
		},
	}
	executor := &mockExecutor{tools: testTools}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(context.Background(), testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Report != "Report with evidence. Severity: WARNING" {
		t.Fatalf("unexpected report: %q", result.Report)
	}

	secondReq := provider.requests[1]
	toolResults := 0
	for _, msg := range secondReq.Messages {
		if msg.Role == llm.RoleTool {
			toolResults++
		}
	}
	if toolResults != 2 {
		t.Fatalf("expected 2 tool results, got %d", toolResults)
	}
}

func TestRun_Timeout(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{Content: "Partial report. Severity: WARNING", FinishReason: llm.FinishReasonStop},
		},
	}
	executor := &mockExecutor{tools: testTools}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(ctx, testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Report != "Partial report. Severity: WARNING" {
		t.Fatalf("unexpected report: %q", result.Report)
	}

	lastReq := provider.requests[0]
	lastMsg := lastReq.Messages[len(lastReq.Messages)-1]
	if lastMsg.Role != llm.RoleUser {
		t.Fatal("expected last message to be user (timeout)")
	}
	if lastMsg.Content != "Time is up. Provide your investigation report now based on what you have gathered so far." {
		t.Fatalf("unexpected timeout message: %q", lastMsg.Content)
	}
	if len(lastReq.Tools) != 0 {
		t.Fatalf("expected no tools in timeout call, got %d", len(lastReq.Tools))
	}
}

func TestRun_MaxIterations(t *testing.T) {
	responses := make([]*llm.ChatResponse, maxIterations+1)
	for i := 0; i < maxIterations; i++ {
		responses[i] = &llm.ChatResponse{
			FinishReason: llm.FinishReasonToolCall,
			ToolCalls: []llm.ToolCall{
				{ID: fmt.Sprintf("call_%d", i), Name: "query_prometheus", Arguments: `{}`},
			},
		}
	}
	responses[maxIterations] = &llm.ChatResponse{
		Content:      "Max iterations reached. Severity: CRITICAL",
		FinishReason: llm.FinishReasonStop,
	}

	provider := &mockProvider{responses: responses}
	executor := &mockExecutor{tools: testTools}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(context.Background(), testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Report != "Max iterations reached. Severity: CRITICAL" {
		t.Fatalf("unexpected report: %q", result.Report)
	}
}

func TestRun_ToolFiltering(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{Content: "Done", FinishReason: llm.FinishReasonStop},
		},
	}
	executor := &mockExecutor{tools: testTools}

	sc := testScenario()
	sc.Tools = []string{"query_prometheus"}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	_, err := ag.Run(context.Background(), testPayload(), sc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := provider.requests[0]
	if len(req.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(req.Tools))
	}
	if req.Tools[0].Name != "query_prometheus" {
		t.Fatalf("expected query_prometheus, got %q", req.Tools[0].Name)
	}
}

func TestRun_CollectsImages(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{
				FinishReason: llm.FinishReasonToolCall,
				ToolCalls: []llm.ToolCall{
					{ID: "call_1", Name: "get_panel_image", Arguments: `{}`},
				},
			},
			{
				Content:      "Report with screenshot. Severity: OK",
				FinishReason: llm.FinishReasonStop,
			},
		},
	}
	executor := &mockExecutor{
		tools:      testTools,
		toolImages: map[string][]byte{"get_panel_image": []byte("fake-png-data")},
	}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(context.Background(), testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(result.Images))
	}
	if string(result.Images[0]) != "fake-png-data" {
		t.Fatalf("unexpected image data: %q", result.Images[0])
	}
}

func TestRun_TimeoutKeepsImages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{
				FinishReason: llm.FinishReasonToolCall,
				ToolCalls: []llm.ToolCall{
					{ID: "call_1", Name: "get_panel_image", Arguments: `{}`},
				},
			},
			{
				Content:      "Timed-out report with screenshot. Severity: WARNING",
				FinishReason: llm.FinishReasonStop,
			},
		},
	}
	// Cancel the context once the first (tool-call) response has been served,
	// forcing the run into timeoutFinish before the second iteration.
	provider.afterCall = func(calls int) {
		if calls == 1 {
			cancel()
		}
	}
	executor := &mockExecutor{
		tools:      testTools,
		toolImages: map[string][]byte{"get_panel_image": []byte("fake-png-data")},
	}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(ctx, testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Fatalf("expected 1 image to survive timeout, got %d", len(result.Images))
	}
	if string(result.Images[0]) != "fake-png-data" {
		t.Fatalf("unexpected image data: %q", result.Images[0])
	}
}

func TestRun_ToolCallError(t *testing.T) {
	provider := &mockProvider{
		responses: []*llm.ChatResponse{
			{
				FinishReason: llm.FinishReasonToolCall,
				ToolCalls: []llm.ToolCall{
					{ID: "call_1", Name: "nonexistent_tool", Arguments: `{}`},
				},
			},
			{
				Content:      "Tool failed, but report is here. Severity: WARNING",
				FinishReason: llm.FinishReasonStop,
			},
		},
	}
	executor := &mockExecutor{
		tools:    testTools,
		failTool: "nonexistent_tool",
	}

	ag := New(provider, executor, testLLMConfig(), testPrompts())
	result, err := ag.Run(context.Background(), testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Report != "Tool failed, but report is here. Severity: WARNING" {
		t.Fatalf("unexpected report: %q", result.Report)
	}

	secondReq := provider.requests[1]
	for _, msg := range secondReq.Messages {
		if msg.Role == llm.RoleTool && msg.ToolCallID == "call_1" {
			if msg.Content[:6] != "error:" {
				t.Fatalf("expected error in tool result, got %q", msg.Content)
			}
		}
	}
}

func TestBuildUserMessage(t *testing.T) {
	msg, err := buildUserMessage(testPayload(), testScenario())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg) == 0 {
		t.Fatal("empty message")
	}

	for _, expected := range []string{"test", "Investigate CPU usage.", "2m0s", "high"} {
		if !contains(msg, expected) {
			t.Fatalf("message missing %q", expected)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
