package agent

import (
	"github.com/nanobotgo/providers"

	"github.com/sashabaranov/go-openai"
)

func ConvertToProviderMessages(messages []map[string]interface{}) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(messages))
	for _, msg := range messages {
		if role, ok := msg["role"].(string); ok {
			content := ""
			if c, ok := msg["content"].(string); ok {
				content = c
			}
			toolCallID := ""
			if t, ok := msg["tool_call_id"].(string); ok {
				toolCallID = t
			}

			openaiMsg := openai.ChatCompletionMessage{
				Role:       role,
				Content:    content,
				ToolCallID: toolCallID,
			}

			if tc, ok := msg["tool_calls"].([]providers.ToolCallRequest); ok {
				openaiMsg.ToolCalls = make([]openai.ToolCall, 0, len(tc))
				for _, t := range tc {
					argsStr, _ := t.Arguments["arguments"].(string)
					openaiMsg.ToolCalls = append(openaiMsg.ToolCalls, openai.ToolCall{
						ID:   t.ID,
						Type: "function",
						Function: openai.FunctionCall{
							Name:      t.Name,
							Arguments: argsStr,
						},
					})
				}
			}

			result = append(result, openaiMsg)
		}
	}
	return result
}

func ConvertToToolDefinitions(definitions []map[string]interface{}) []providers.ToolDefinition {
	result := make([]providers.ToolDefinition, 0, len(definitions))
	for _, def := range definitions {
		if funcDef, ok := def["function"].(map[string]interface{}); ok {
			name := ""
			if n, ok := funcDef["name"].(string); ok {
				name = n
			}
			description := ""
			if d, ok := funcDef["description"].(string); ok {
				description = d
			}
			params := map[string]interface{}{}
			if p, ok := funcDef["parameters"].(map[string]interface{}); ok {
				params = p
			}
			result = append(result, providers.ToolDefinition{
				Type: "function",
				Function: providers.ToolFunction{
					Name:        name,
					Description: description,
					Parameters:  params,
				},
			})
		}
	}
	return result
}
