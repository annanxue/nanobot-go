package providers

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

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

	// Check if content contains JSON code block with tool call
	if len(toolCalls) == 0 && choice.Message.Content != "" {
		if containsJSONCodeBlock(choice.Message.Content) {
			jsonContent := extractJSONFromCodeBlock(choice.Message.Content)
			if jsonContent != "" {
				parsedToolCalls, err := parseToolCallFromJSON(jsonContent)
				if err == nil && len(parsedToolCalls) > 0 {
					toolCalls = parsedToolCalls
					// Clear content since we're treating this as a tool call
					choice.Message.Content = ""
				}
			}
		}
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

// containsJSONCodeBlock checks if the content contains a JSON code block
func containsJSONCodeBlock(content string) bool {
	pattern := regexp.MustCompile("```json[\\s\\S]*?```")
	return pattern.MatchString(content)
}

// extractJSONFromCodeBlock extracts JSON content from a code block
func extractJSONFromCodeBlock(content string) string {
	pattern := regexp.MustCompile("```json[\\s\\S]*?```")
	matches := pattern.FindString(content)
	if matches == "" {
		return ""
	}
	// Remove the code block markers
	jsonContent := strings.TrimPrefix(matches, "```json")
	jsonContent = strings.TrimSuffix(jsonContent, "```")
	jsonContent = strings.TrimSpace(jsonContent)
	return jsonContent
}

// parseToolCallFromJSON parses tool call from JSON content
func parseToolCallFromJSON(jsonContent string) ([]ToolCallRequest, error) {
	var toolCallData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &toolCallData); err != nil {
		return nil, err
	}

	// Check if it has tool call structure
	if toolName, ok := toolCallData["tool"].(string); ok {
		// Standard tool call format: {"tool": "name", "params": {...}}
		params := make(map[string]interface{})
		if p, ok := toolCallData["params"].(map[string]interface{}); ok {
			params = p
		} else if p, ok := toolCallData["parameters"].(map[string]interface{}); ok {
			params = p
		} else {
			// Use the entire object as params if no specific params field
			for k, v := range toolCallData {
				if k != "tool" {
					params[k] = v
				}
			}
		}

		return []ToolCallRequest{
			{
				ID:        "auto-generated",
				Name:      toolName,
				Arguments: params,
			},
		}, nil
	} else if _, ok := toolCallData["type"].(string); ok {
		// Interaction tool format: {"type": "click", "x": 100, "y": 200}
		return []ToolCallRequest{
			{
				ID:        "auto-generated",
				Name:      "interaction",
				Arguments: toolCallData,
			},
		}, nil
	}

	return nil, nil
}
