// Package anthropic implements the llm.Provider interface on top of the
// official Anthropic Messages API (Claude models).
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/extended-log-go/log"
)

// defaultModel is used when no model is configured.
const defaultModel = string(anthropic.ModelClaudeOpus4_8)

// fallbackMaxTokens bounds a single response when the caller does not set one.
const fallbackMaxTokens = 16000

type Provider struct {
	client          *anthropic.Client
	model           string
	reasoningEffort string
}

// New creates a Claude provider. An empty model falls back to defaultModel.
// A non-empty reasoningEffort (low|medium|high|xhigh|max) enables adaptive
// thinking with that effort level.
func New(apiKey, baseURL, model, reasoningEffort string) *Provider {
	if model == "" {
		model = defaultModel
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := anthropic.NewClient(opts...)

	return &Provider{
		client:          &client,
		model:           model,
		reasoningEffort: reasoningEffort,
	}
}

func (p *Provider) ChatCompletion(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages, err := convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	maxTokens := int64(req.MaxTokens)
	if maxTokens <= 0 {
		maxTokens = fallbackMaxTokens
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		Messages:  messages,
	}
	if req.SystemPrompt != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.SystemPrompt}}
	}
	if len(req.Tools) > 0 {
		params.Tools = convertTools(req.Tools)
	}
	if p.reasoningEffort != "" {
		params.Thinking = anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		}
		params.OutputConfig = anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffort(p.reasoningEffort),
		}
	}

	if reqJSON, err := json.Marshal(params); err == nil {
		log.Tracef("[anthropic] request: %s", string(reqJSON))
	}

	// Stream and accumulate: large max_tokens values risk HTTP timeouts on a
	// single non-streaming call, so we always stream and rebuild the message.
	stream := p.client.Messages.NewStreaming(ctx, params)
	message := anthropic.Message{}
	for stream.Next() {
		if err := message.Accumulate(stream.Current()); err != nil {
			return nil, fmt.Errorf("accumulate stream event: %w", err)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("messages stream: %w", err)
	}

	resp := &llm.ChatResponse{
		FinishReason: mapStopReason(message.StopReason),
		PromptTokens: int(message.Usage.InputTokens),
	}

	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.TextBlock:
			resp.Content += variant.Text
		case anthropic.ToolUseBlock:
			resp.ToolCalls = append(resp.ToolCalls, llm.ToolCall{
				ID:        variant.ID,
				Name:      variant.Name,
				Arguments: string(variant.Input),
			})
		case anthropic.ThinkingBlock:
			resp.Thinking = append(resp.Thinking, llm.ThinkingBlock{
				Text:      variant.Thinking,
				Signature: variant.Signature,
			})
		case anthropic.RedactedThinkingBlock:
			resp.Thinking = append(resp.Thinking, llm.ThinkingBlock{
				Redacted: true,
				Data:     variant.Data,
			})
		}
	}

	if respJSON, err := json.Marshal(message); err == nil {
		log.Tracef("[anthropic] response: %s", string(respJSON))
	}

	return resp, nil
}

// convertMessages maps provider-agnostic messages to the Anthropic shape.
// The Messages API requires alternating user/assistant turns and groups all
// tool results from one turn into a single user message, so consecutive
// RoleTool messages are coalesced into one user message of tool_result blocks.
func convertMessages(messages []llm.Message) ([]anthropic.MessageParam, error) {
	var out []anthropic.MessageParam

	for i := 0; i < len(messages); {
		msg := messages[i]
		switch msg.Role {
		case llm.RoleUser:
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			i++

		case llm.RoleAssistant:
			var blocks []anthropic.ContentBlockParamUnion
			// Thinking blocks must come first: when thinking is enabled the
			// assistant turn that calls tools has to start with them, with
			// their signatures preserved, or the API rejects the next request.
			for _, tb := range msg.Thinking {
				if tb.Redacted {
					blocks = append(blocks, anthropic.ContentBlockParamUnion{
						OfRedactedThinking: &anthropic.RedactedThinkingBlockParam{Data: tb.Data},
					})
				} else {
					blocks = append(blocks, anthropic.ContentBlockParamUnion{
						OfThinking: &anthropic.ThinkingBlockParam{Thinking: tb.Text, Signature: tb.Signature},
					})
				}
			}
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				var input any
				if tc.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Arguments), &input); err != nil {
						return nil, fmt.Errorf("unmarshal tool call arguments for %q: %w", tc.Name, err)
					}
				}
				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: input,
					},
				})
			}
			out = append(out, anthropic.NewAssistantMessage(blocks...))
			i++

		case llm.RoleTool:
			var blocks []anthropic.ContentBlockParamUnion
			for i < len(messages) && messages[i].Role == llm.RoleTool {
				content := messages[i].Content
				if content == "" {
					content = "(tool returned no text content)"
				}
				blocks = append(blocks, anthropic.NewToolResultBlock(messages[i].ToolCallID, content, false))
				i++
			}
			out = append(out, anthropic.NewUserMessage(blocks...))

		default:
			return nil, fmt.Errorf("unknown role: %s", msg.Role)
		}
	}

	return out, nil
}

func convertTools(tools []llm.Tool) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		schema := anthropic.ToolInputSchemaParam{}
		if t.Parameters != nil {
			schema.Properties = t.Parameters["properties"]
			schema.Required = toStringSlice(t.Parameters["required"])
		}
		out[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: schema,
			},
		}
	}
	return out
}

func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func mapStopReason(r anthropic.StopReason) llm.FinishReason {
	if r == anthropic.StopReasonToolUse {
		return llm.FinishReasonToolCall
	}
	return llm.FinishReasonStop
}
