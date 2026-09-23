package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tsisar/alert-agent/internal/llm"
)

type imageTestExecutor struct {
	calls int
	run   func(int, string, json.RawMessage) (*llm.ToolResult, error)
}

func (e *imageTestExecutor) Tools() []llm.Tool { return testTools }
func (e *imageTestExecutor) CallTool(_ context.Context, name string, args json.RawMessage) (*llm.ToolResult, error) {
	e.calls++
	return e.run(e.calls, name, args)
}

func TestImageExecutor_ReusesPanelWithReorderedArguments(t *testing.T) {
	underlying := &imageTestExecutor{run: func(int, string, json.RawMessage) (*llm.ToolResult, error) {
		return &llm.ToolResult{Content: "https://grafana/panel", Images: [][]byte{[]byte("png")}}, nil
	}}
	executor := newImageExecutor(underlying, true)
	args := []string{`{"dashboardUid":"a","panelId":2,"variables":{"var-host":"one","var-ds":"p"}}`, ` {"variables":{"var-ds":"p","var-host":"one"}, "panelId":2,"dashboardUid":"a"} `}
	for i, arg := range args {
		result, err := executor.CallTool(context.Background(), "grafana__get_panel_image", json.RawMessage(arg))
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Images) != 1-i {
			t.Fatalf("call %d images=%d", i, len(result.Images))
		}
		if !strings.Contains(result.Content, "https://grafana/panel") || !strings.Contains(result.Content, imageNoteAttached) {
			t.Fatalf("missing result/confirmation: %s", result.Content)
		}
	}
	if underlying.calls != 1 {
		t.Fatalf("renders=%d", underlying.calls)
	}
}

func TestImageExecutor_RetriesUntilImageReturned(t *testing.T) {
	underlying := &imageTestExecutor{run: func(call int, _ string, _ json.RawMessage) (*llm.ToolResult, error) {
		switch call {
		case 1:
			return nil, errors.New("render failed")
		case 2:
			return &llm.ToolResult{Content: "error: no image"}, nil
		case 3:
			return &llm.ToolResult{Images: [][]byte{nil, {}}}, nil
		default:
			return &llm.ToolResult{Images: [][]byte{[]byte("png")}}, nil
		}
	}}
	executor := newImageExecutor(underlying, true)
	for i := 1; i <= 5; i++ {
		result, err := executor.CallTool(context.Background(), "get_panel_image", json.RawMessage(`{}`))
		if i == 1 {
			if err == nil {
				t.Fatal("expected error")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if i == 4 {
			want = 1
		}
		if len(result.Images) != want {
			t.Fatalf("attempt %d: images=%d", i, len(result.Images))
		}
	}
	if underlying.calls != 4 {
		t.Fatalf("renders=%d", underlying.calls)
	}
}

func TestImageExecutor_DistinctViewsAndTools(t *testing.T) {
	calls := []struct{ name, args string }{
		{"grafana__get_panel_image", `{"dashboardUid":"a","panelId":2}`},
		{"grafana__get_panel_image", `{"dashboardUid":"a","panelId":3}`},
		{"grafana__get_panel_image", `{"dashboardUid":"b","panelId":2}`},
		{"grafana__get_panel_image", `{"dashboardUid":"a","panelId":2,"variables":{"var-host":"two"}}`},
		{"grafana__get_panel_image", `{"dashboardUid":"a","panelId":2,"timeRange":{"from":"now-3h","to":"now"}}`},
		{"other__get_panel_image", `{"dashboardUid":"a","panelId":2}`},
	}
	underlying := &imageTestExecutor{run: func(call int, _ string, _ json.RawMessage) (*llm.ToolResult, error) {
		return &llm.ToolResult{Images: [][]byte{{byte(call)}}}, nil
	}}
	executor := newImageExecutor(underlying, true)
	for _, call := range calls {
		result, err := executor.CallTool(context.Background(), call.name, json.RawMessage(call.args))
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Images) != 1 {
			t.Fatalf("view lost: %+v", call)
		}
	}
	if underlying.calls != len(calls) {
		t.Fatalf("renders=%d", underlying.calls)
	}
}

func TestImageExecutor_DeduplicatesContentWithoutCachingOtherTools(t *testing.T) {
	original := &llm.ToolResult{Images: [][]byte{[]byte("one"), []byte("one"), []byte("two")}}
	underlying := &imageTestExecutor{run: func(int, string, json.RawMessage) (*llm.ToolResult, error) { return original, nil }}
	executor := newImageExecutor(underlying, true)
	for i := 0; i < 2; i++ {
		result, err := executor.CallTool(context.Background(), "browser_screenshot", json.RawMessage(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		want := 2
		if i == 1 {
			want = 0
		}
		if len(result.Images) != want {
			t.Fatalf("images=%d", len(result.Images))
		}
	}
	if underlying.calls != 2 || len(original.Images) != 3 {
		t.Fatal("unexpected caching or mutation")
	}
}

func TestRun_ImageCacheIsPerInvestigation(t *testing.T) {
	provider := &mockProvider{}
	for run := 0; run < 2; run++ {
		provider.responses = append(provider.responses,
			&llm.ChatResponse{FinishReason: llm.FinishReasonToolCall, ToolCalls: []llm.ToolCall{
				{ID: "first", Name: "get_panel_image", Arguments: `{"panelId":2}`},
				{ID: "same-iteration", Name: "get_panel_image", Arguments: `{"panelId":2}`},
			}},
			&llm.ChatResponse{FinishReason: llm.FinishReasonToolCall, ToolCalls: []llm.ToolCall{{ID: "next-iteration", Name: "get_panel_image", Arguments: `{"panelId":2}`}}},
			&llm.ChatResponse{FinishReason: llm.FinishReasonStop, Content: "Report"},
			&llm.ChatResponse{FinishReason: llm.FinishReasonStop, Content: "Summary"},
		)
	}
	underlying := &imageTestExecutor{run: func(int, string, json.RawMessage) (*llm.ToolResult, error) {
		return &llm.ToolResult{Images: [][]byte{[]byte("png")}}, nil
	}}
	ag := New(provider, underlying, testLLMConfig(), testPrompts())
	for run := 0; run < 2; run++ {
		result, err := ag.Run(context.Background(), testPayload(), testScenario())
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Images) != 1 {
			t.Fatalf("run %d: images=%d", run, len(result.Images))
		}
		messages := provider.requests[run*4+2].Messages
		for _, id := range []string{"first", "same-iteration", "next-iteration"} {
			found := false
			for _, m := range messages {
				if m.Role == llm.RoleTool && m.ToolCallID == id && strings.Contains(m.Content, "Image captured") {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing tool response for %s", id)
			}
		}
	}
	if underlying.calls != 2 {
		t.Fatalf("renders across two investigations=%d", underlying.calls)
	}
}

func TestImageExecutor_NoteFollowsScenarioDelivery(t *testing.T) {
	underlying := &imageTestExecutor{run: func(int, string, json.RawMessage) (*llm.ToolResult, error) {
		return &llm.ToolResult{Content: "panel", Images: [][]byte{[]byte("png")}}, nil
	}}
	for _, tc := range []struct {
		sendImages bool
		want       string
	}{{true, imageNoteAttached}, {false, imageNoteNotSent}} {
		result, err := newImageExecutor(underlying, tc.sendImages).CallTool(context.Background(), "get_panel_image", json.RawMessage(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		if result.Content != "panel"+tc.want {
			t.Fatalf("sendImages=%v: content=%q", tc.sendImages, result.Content)
		}
	}
}
