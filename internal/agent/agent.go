package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tsisar/alert-agent/internal/config"
	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/alert-agent/internal/model"
	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/storage"
	"github.com/tsisar/extended-log-go/log"
)

const (
	maxIterations            = 20
	maxToolCallsPerIteration = 10
	maxSummaryLen            = 500
	trimmedPlaceholder       = "[trimmed]"
)

// Result holds the output of an agent investigation.
type Result struct {
	Summary string
	Report  string
	Images  [][]byte
}

// Agent orchestrates an LLM tool-use loop to investigate a Grafana alert.
type Agent struct {
	provider llm.Provider
	executor ToolExecutor
	llmCfg   config.LLMConfig
	prompts  storage.PromptRepository
}

func New(provider llm.Provider, executor ToolExecutor, llmCfg config.LLMConfig, prompts storage.PromptRepository) *Agent {
	return &Agent{
		provider: provider,
		executor: executor,
		llmCfg:   llmCfg,
		prompts:  prompts,
	}
}

// Run performs the investigation: builds a prompt from the alert and scenario,
// enters the tool-use loop, and returns the final report.
func (a *Agent) Run(ctx context.Context, payload *model.WebhookPayload, sc *scenario.Scenario) (*Result, error) {
	prompts, err := a.prompts.GetAll()
	if err != nil {
		return nil, fmt.Errorf("load prompts: %w", err)
	}

	tools := a.toolsForScenario(sc)
	toolDefs := tools.Tools()

	log.Debugf("[agent] scenario=%q tools=%d timeout=%s priority=%s",
		sc.Name, len(toolDefs), sc.Timeout.Duration, sc.Priority)

	toolNames := make([]string, len(toolDefs))
	for i, t := range toolDefs {
		toolNames[i] = t.Name
	}
	log.Debugf("[agent] available tools: %v", toolNames)

	userContent, err := buildUserMessage(payload, sc)
	if err != nil {
		return nil, fmt.Errorf("build user message: %w", err)
	}

	log.Debugf("[agent] user message length: %d chars", len(userContent))
	log.Tracef("[agent] user message:\n%s", userContent)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: userContent},
	}

	var images [][]byte
	var lastPromptTokens int

	for i := 0; i < maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			log.Debugf("[agent] context cancelled before iteration %d, finishing", i)
			return a.timeoutFinish(ctx, prompts, messages)
		}

		if a.llmCfg.ContextLimit > 0 && lastPromptTokens > 0 {
			messages = a.trimMessages(messages, lastPromptTokens)
		}

		log.Debugf("[agent] iteration %d: sending %d messages to LLM", i+1, len(messages))

		resp, err := a.provider.ChatCompletion(ctx, &llm.ChatRequest{
			SystemPrompt: prompts.System,
			Messages:     messages,
			Tools:        toolDefs,
			MaxTokens:    a.llmCfg.MaxTokens,
		})
		if err != nil {
			if ctx.Err() != nil {
				log.Debugf("[agent] LLM call failed due to timeout, finishing")
				return a.timeoutFinish(context.WithoutCancel(ctx), prompts, messages)
			}
			return nil, fmt.Errorf("chat completion (iteration %d): %w", i, err)
		}

		lastPromptTokens = resp.PromptTokens

		log.Debugf("[agent] LLM response: finish_reason=%s tool_calls=%d content_length=%d prompt_tokens=%d",
			resp.FinishReason, len(resp.ToolCalls), len(resp.Content), resp.PromptTokens)

		if resp.Content != "" {
			log.Tracef("[agent] LLM content: %s", truncate(resp.Content, 500))
		}

		if resp.FinishReason == llm.FinishReasonStop || len(resp.ToolCalls) == 0 {
			log.Infof("agent finished: iterations=%d report_length=%d", i+1, len(resp.Content))
			summary := a.summarize(ctx, prompts, resp.Content)
			return &Result{Summary: summary, Report: resp.Content, Images: images}, nil
		}

		toolCalls := resp.ToolCalls
		if len(toolCalls) > maxToolCallsPerIteration {
			log.Warnf("[agent] too many tool calls (%d), limiting to %d", len(toolCalls), maxToolCallsPerIteration)
			toolCalls = toolCalls[:maxToolCallsPerIteration]
		}

		messages = append(messages, llm.Message{
			Role:      llm.RoleAssistant,
			ToolCalls: toolCalls,
		})

		for _, tc := range toolCalls {
			log.Infof("tool call: name=%s id=%s", tc.Name, tc.ID)
			log.Debugf("[agent] tool args: %s", truncate(tc.Arguments, 300))

			toolResult, callErr := tools.CallTool(ctx, tc.Name, json.RawMessage(tc.Arguments))

			var content string
			if callErr != nil {
				log.Errorf("tool call failed: name=%s err=%v", tc.Name, callErr)
				content = fmt.Sprintf("error: %v", callErr)
			} else {
				content = toolResult.Content
				if len(toolResult.Images) > 0 {
					images = append(images, toolResult.Images...)
					log.Debugf("[agent] collected %d image(s) from tool %s", len(toolResult.Images), tc.Name)
				}
			}

			log.Debugf("[agent] tool result: name=%s length=%d preview=%s",
				tc.Name, len(content), truncate(content, 200))

			messages = append(messages, llm.Message{
				Role:       llm.RoleTool,
				Content:    content,
				ToolCallID: tc.ID,
			})
		}
	}

	log.Warnf("agent reached max iterations (%d), forcing finish", maxIterations)
	return a.timeoutFinish(ctx, prompts, messages)
}

// timeoutFinish injects a "wrap up" message and does one final LLM call without tools.
func (a *Agent) timeoutFinish(ctx context.Context, prompts *storage.PromptSet, messages []llm.Message) (*Result, error) {
	log.Debugf("[agent] timeout finish: injecting wrap-up message, total messages=%d", len(messages)+1)

	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: "Time is up. Provide your investigation report now based on what you have gathered so far.",
	})

	resp, err := a.provider.ChatCompletion(ctx, &llm.ChatRequest{
		SystemPrompt: prompts.System,
		Messages:     messages,
		MaxTokens:    a.llmCfg.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("timeout finish call: %w", err)
	}

	log.Debugf("[agent] timeout finish report length: %d", len(resp.Content))
	summary := a.summarize(ctx, prompts, resp.Content)
	return &Result{Summary: summary, Report: resp.Content}, nil
}

func (a *Agent) toolsForScenario(sc *scenario.Scenario) ToolExecutor {
	if len(sc.Tools) == 0 {
		return a.executor
	}
	return NewFilteredExecutor(a.executor, sc.Tools)
}

func buildUserMessage(payload *model.WebhookPayload, sc *scenario.Scenario) (string, error) {
	alertJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal alert payload: %w", err)
	}

	msg := fmt.Sprintf("## Alert\n\n```json\n%s\n```\n\n## Scenario: %s\n\n%s\n\nTime budget: %s. Priority: %s.",
		string(alertJSON),
		sc.Name,
		sc.Prompt,
		sc.Timeout.String(),
		sc.Priority,
	)

	return msg, nil
}

// ResolvedPrompt returns the resolved alert notification prompt.
func (a *Agent) ResolvedPrompt() string {
	s, _ := a.prompts.Get("resolved")
	return s
}

// PausedPrompt returns the paused alert notification prompt.
func (a *Agent) PausedPrompt() string {
	s, _ := a.prompts.Get("paused")
	return s
}

// Notify makes a single LLM call to produce a short notification message
// for non-firing alerts (resolved, paused). No investigation is performed.
func (a *Agent) Notify(ctx context.Context, payload *model.WebhookPayload, prompt string) (string, error) {
	alertJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal alert payload: %w", err)
	}

	resp, err := a.provider.ChatCompletion(ctx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt + string(alertJSON)},
		},
		MaxTokens: a.llmCfg.SummaryMaxToks,
	})
	if err != nil {
		return "", fmt.Errorf("notify LLM call: %w", err)
	}

	return resp.Content, nil
}

// summarize makes a separate LLM call to produce a short summary from the report.
func (a *Agent) summarize(ctx context.Context, prompts *storage.PromptSet, report string) string {
	log.Debugf("[agent] generating summary from report (%d chars)", len(report))

	resp, err := a.provider.ChatCompletion(ctx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompts.Summary + report},
		},
		MaxTokens: a.llmCfg.SummaryMaxToks,
	})
	if err != nil {
		log.Errorf("[agent] summary generation failed: %v", err)
		return ""
	}

	summary := resp.Content
	if len(summary) > maxSummaryLen {
		summary = summary[:maxSummaryLen]
		if idx := lastIndexByte(summary, '.'); idx > 0 {
			summary = summary[:idx+1]
		}
		log.Warnf("[agent] summary truncated to %d chars", len(summary))
	}

	log.Debugf("[agent] summary generated: %s", summary)
	return summary
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// trimMessages reduces context size by replacing the content of older tool-result
// messages with a placeholder. It uses the prompt token count from the previous
// API response plus an estimate for newly added messages to decide whether trimming
// is needed.
func (a *Agent) trimMessages(messages []llm.Message, lastPromptTokens int) []llm.Message {
	// Estimate tokens added since the last API call: every message after
	// the last assistant response that was already counted.
	var newChars int
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role == llm.RoleAssistant && len(msg.ToolCalls) > 0 {
			// count this assistant message and all tool results after it
			for _, tc := range msg.ToolCalls {
				newChars += len(tc.Arguments)
			}
			for j := i + 1; j < len(messages); j++ {
				newChars += len(messages[j].Content)
			}
			break
		}
	}

	estimatedTokens := lastPromptTokens + newChars/3
	budget := a.llmCfg.ContextLimit - a.llmCfg.MaxTokens

	if estimatedTokens <= budget {
		return messages
	}

	log.Warnf("[agent] context estimate %d exceeds budget %d, trimming old tool results", estimatedTokens, budget)

	// Trim oldest tool results first, skip the first message (user prompt)
	// and the last group of messages (most recent assistant + tool results).
	for i := 1; i < len(messages); i++ {
		if messages[i].Role != llm.RoleTool || messages[i].Content == trimmedPlaceholder {
			continue
		}

		saved := len(messages[i].Content) / 3
		messages[i].Content = trimmedPlaceholder
		estimatedTokens -= saved

		log.Debugf("[agent] trimmed message %d, saved ~%d tokens, estimate now %d", i, saved, estimatedTokens)

		if estimatedTokens <= budget {
			break
		}
	}

	return messages
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
