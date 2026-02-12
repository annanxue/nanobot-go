package agent

import (
	"github.com/nanobotgo/providers"
)

func ConvertToProviderMessages(messages []map[string]interface{}) []providers.Message {
	result := make([]providers.Message, 0, len(messages))
	for _, msg := range messages {
		if role, ok := msg["role"].(string); ok {
			content := ""
			if c, ok := msg["content"].(string); ok {
				content = c
			}
			result = append(result, providers.Message{
				Role:    role,
				Content: content,
			})
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