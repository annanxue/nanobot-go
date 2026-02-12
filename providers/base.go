package providers

import "context"

type ToolCallRequest struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type LLMResponse struct {
	Content          string            `json:"content"`
	ToolCalls        []ToolCallRequest `json:"tool_calls,omitempty"`
	FinishReason     string            `json:"finish_reason"`
	Usage            map[string]int    `json:"usage,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
}

func (r *LLMResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

type Message struct {
	Role       string            `json:"role"`
	Content    string            `json:"content"`
	ToolCalls  []ToolCallRequest `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"`
}

type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type LLMProvider interface {
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, maxTokens int, temperature float64) (*LLMResponse, error)
	GetDefaultModel() string
}
