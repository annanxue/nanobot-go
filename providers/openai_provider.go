package providers

import (
	"context"
	"encoding/json"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client       *openai.Client
	defaultModel string
}

func NewOpenAIProvider(apiKey, apiBase, defaultModel string, extraHeaders map[string]string, providerName string) *OpenAIProvider {
	config := openai.DefaultConfig(apiKey)
	if apiBase != "" {
		config.BaseURL = apiBase
	}
	client := openai.NewClientWithConfig(config)

	return &OpenAIProvider{
		client:       client,
		defaultModel: defaultModel,
	}
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []ToolDefinition, model string, maxTokens int, temperature float64) (*LLMResponse, error) {
	if model == "" {
		model = p.defaultModel
	}
	var openaiTools []openai.Tool
	if len(tools) > 0 {
		openaiTools = make([]openai.Tool, 0, len(tools))
		for _, tool := range tools {
			openaiTools = append(openaiTools, openai.Tool{
				Type: "function",
				Function: &openai.FunctionDefinition{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Tools:    openaiTools,
		// MaxTokens:   maxTokens,
		Temperature: float32(temperature),
	}

	if len(tools) > 0 {
		req.ToolChoice = "auto"
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return &LLMResponse{
			Content:      "No response from LLM",
			FinishReason: "stop",
		}, nil
	}

	choice := resp.Choices[0]
	toolCalls := make([]ToolCallRequest, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		argsMap := make(map[string]interface{})
		if tc.Function.Arguments != "" {
			json.Unmarshal([]byte(tc.Function.Arguments), &argsMap)
		}
		toolCalls = append(toolCalls, ToolCallRequest{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: argsMap,
		})
	}

	finishReason := string(choice.FinishReason)
	response := &LLMResponse{
		Content:          choice.Message.Content,
		ToolCalls:        toolCalls,
		FinishReason:     finishReason,
		ReasoningContent: choice.Message.ReasoningContent,
	}

	if resp.Usage.TotalTokens > 0 {
		response.Usage = map[string]int{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		}
	}

	return response, nil
}

func (p *OpenAIProvider) GetDefaultModel() string {
	return p.defaultModel
}
