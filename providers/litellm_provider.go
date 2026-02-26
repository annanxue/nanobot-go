package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"

	"github.com/voocel/litellm"
	"github.com/voocel/litellm/providers"
)

type LiteLLMProvider struct {
	client       *litellm.Client
	defaultModel string
	providerName string
}

func NewLiteLLMProvider(apiKey, apiBase, defaultModel string, extraHeaders map[string]string, providerName string) *LiteLLMProvider {
	gatewaySpec := FindGateway(providerName, apiKey, apiBase)

	providerCfg := providers.ProviderConfig{
		APIKey:  apiKey,
		BaseURL: apiBase,
	}

	var client *litellm.Client
	var err error

	if gatewaySpec != nil && gatewaySpec.LiteLLMPrefix != "" {
		providerType := gatewaySpec.LiteLLMPrefix
		client, err = litellm.NewWithProvider(providerType, providerCfg)
	} else {
		client, err = litellm.New(providers.NewOpenAI(providerCfg))
	}

	if err != nil {
		panic(fmt.Sprintf("failed to create litellm client: %v", err))
	}

	return &LiteLLMProvider{
		client:       client,
		defaultModel: defaultModel,
		providerName: providerName,
	}
}

func (p *LiteLLMProvider) Chat(ctx context.Context, messages []openai.ChatCompletionMessage, tools []ToolDefinition, model string, maxTokens int, temperature float64) (*LLMResponse, error) {
	if model == "" {
		model = p.defaultModel
	}
	providerMsgs := convertOpenAIMessagesToLitellm(messages)

	req := litellm.NewRequestWithMessages(model, providerMsgs,
		litellm.WithMaxTokens(maxTokens),
		litellm.WithTemperature(temperature),
	)

	if len(tools) > 0 {
		litellmTools := make([]litellm.Tool, 0, len(tools))
		for _, tool := range tools {
			litellmTools = append(litellmTools, litellm.Tool{
				Type: "function",
				Function: providers.FunctionDef{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  tool.Function.Parameters,
				},
			})
		}
		req.Tools = litellmTools
		req.ToolChoice = "auto"
	}

	resp, err := p.client.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	return convertToLLMResponse(resp), nil
}

func (p *LiteLLMProvider) GetDefaultModel() string {
	return p.defaultModel
}

func convertOpenAIMessagesToLitellm(messages []openai.ChatCompletionMessage) []providers.Message {
	result := make([]providers.Message, 0, len(messages))
	for _, msg := range messages {
		litellmMsg := providers.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]providers.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				toolCalls = append(toolCalls, providers.ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: providers.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
			litellmMsg.ToolCalls = toolCalls
		}

		result = append(result, litellmMsg)
	}
	return result
}
func convertToLLMResponse(resp *providers.Response) *LLMResponse {
	toolCalls := make([]ToolCallRequest, 0, len(resp.ToolCalls))
	for _, tc := range resp.ToolCalls {
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

	var reasoningContent string
	if resp.Reasoning != nil {
		reasoningContent = resp.Reasoning.Content
	}

	return &LLMResponse{
		Content:          resp.Content,
		ToolCalls:        toolCalls,
		FinishReason:     resp.FinishReason,
		ReasoningContent: reasoningContent,
		Usage: map[string]int{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		},
	}
}
