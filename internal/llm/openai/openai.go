package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/extended-log-go/log"
)

type Provider struct {
	client *openai.Client
	model  string
}

func New(apiKey, baseURL, model string) *Provider {
	if model == "" {
		model = "gpt-4o"
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	return &Provider{
		client: &client,
		model:  model,
	}
}

func (p *Provider) ChatCompletion(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages, err := convertMessages(req.SystemPrompt, req.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(model),
		Messages: messages,
	}

	if len(req.Tools) > 0 {
		params.Tools = convertTools(req.Tools)
	}

	if req.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(req.MaxTokens))
	}

	// Trace: log full request
	if reqJSON, err := json.Marshal(params); err == nil {
		log.Tracef("[openai] request: %s", string(reqJSON))
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("chat completion: empty response, no choices returned")
	}

	choice := completion.Choices[0]

	resp := &llm.ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: mapFinishReason(choice.FinishReason),
		PromptTokens: int(completion.Usage.PromptTokens),
	}

	for _, tc := range choice.Message.ToolCalls {
		resp.ToolCalls = append(resp.ToolCalls, llm.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	// Trace: log full response
	if respJSON, err := json.Marshal(completion); err == nil {
		log.Tracef("[openai] response: %s", string(respJSON))
	}

	return resp, nil
}

func convertMessages(systemPrompt string, messages []llm.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	var out []openai.ChatCompletionMessageParamUnion

	if systemPrompt != "" {
		out = append(out, openai.DeveloperMessage(systemPrompt))
	}

	for _, msg := range messages {
		param, err := convertMessage(msg)
		if err != nil {
			return nil, err
		}
		out = append(out, param)
	}

	return out, nil
}

func convertMessage(msg llm.Message) (openai.ChatCompletionMessageParamUnion, error) {
	switch msg.Role {
	case llm.RoleUser:
		return openai.UserMessage(msg.Content), nil

	case llm.RoleAssistant:
		if len(msg.ToolCalls) == 0 {
			return openai.AssistantMessage(msg.Content), nil
		}
		toolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			toolCalls[i] = openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				},
			}
		}
		return openai.ChatCompletionMessageParamUnion{
			OfAssistant: &openai.ChatCompletionAssistantMessageParam{
				ToolCalls: toolCalls,
			},
		}, nil

	case llm.RoleTool:
		return openai.ToolMessage(msg.Content, msg.ToolCallID), nil

	default:
		return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("unknown role: %s", msg.Role)
	}
}

func convertTools(tools []llm.Tool) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, len(tools))
	for i, t := range tools {
		out[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  openai.FunctionParameters(t.Parameters),
		})
	}
	return out
}

func mapFinishReason(reason string) llm.FinishReason {
	switch reason {
	case "tool_calls":
		return llm.FinishReasonToolCall
	default:
		return llm.FinishReasonStop
	}
}
