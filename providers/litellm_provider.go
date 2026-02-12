package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type LiteLLMProvider struct {
	apiKey       string
	apiBase      string
	defaultModel string
	extraHeaders map[string]string
	gateway      *ProviderSpec
}

func NewLiteLLMProvider(apiKey, apiBase, defaultModel string, extraHeaders map[string]string, providerName string) *LiteLLMProvider {
	gateway := FindGateway(providerName, apiKey, apiBase)

	if apiKey != "" {
		setupEnv(apiKey, apiBase, defaultModel, gateway)
	}

	return &LiteLLMProvider{
		apiKey:       apiKey,
		apiBase:      apiBase,
		defaultModel: defaultModel,
		extraHeaders: extraHeaders,
		gateway:      gateway,
	}
}

func setupEnv(apiKey, apiBase, model string, gateway *ProviderSpec) {
	spec := gateway
	if spec == nil {
		spec = FindByModel(model)
	}
	if spec == nil {
		return
	}

	if gateway != nil {
		os.Setenv(spec.EnvKey, apiKey)
	} else {
		if os.Getenv(spec.EnvKey) == "" {
			os.Setenv(spec.EnvKey, apiKey)
		}
	}

	effectiveBase := apiBase
	if effectiveBase == "" {
		effectiveBase = spec.DefaultAPIBase
	}

	for _, envExtra := range spec.EnvExtras {
		resolved := strings.ReplaceAll(envExtra.Value, "{api_key}", apiKey)
		resolved = strings.ReplaceAll(resolved, "{api_base}", effectiveBase)
		if os.Getenv(envExtra.Name) == "" {
			os.Setenv(envExtra.Name, resolved)
		}
	}
}

func (p *LiteLLMProvider) resolveModel(model string) string {
	if p.gateway != nil {
		prefix := p.gateway.LiteLLMPrefix
		if p.gateway.StripModelPrefix {
			parts := strings.Split(model, "/")
			model = parts[len(parts)-1]
		}
		if prefix != "" && !strings.HasPrefix(model, prefix+"/") {
			model = prefix + "/" + model
		}
		return model
	}

	spec := FindByModel(model)
	if spec != nil && spec.LiteLLMPrefix != "" {
		hasPrefix := false
		for _, skip := range spec.SkipPrefixes {
			if strings.HasPrefix(model, skip) {
				hasPrefix = true
				break
			}
		}
		if !hasPrefix {
			model = spec.LiteLLMPrefix + "/" + model
		}
	}

	return model
}

func (p *LiteLLMProvider) applyModelOverrides(model string, kwargs map[string]interface{}) {
	modelLower := strings.ToLower(model)
	spec := FindByModel(model)
	if spec == nil {
		return
	}

	for _, override := range spec.ModelOverrides {
		if strings.Contains(modelLower, override.Pattern) {
			for k, v := range override.Overrides {
				kwargs[k] = v
			}
			return
		}
	}
}

type ChatRequest struct {
	Model        string            `json:"model"`
	Messages     []Message         `json:"messages"`
	MaxTokens    int               `json:"max_tokens,omitempty"`
	Temperature  float64           `json:"temperature,omitempty"`
	Tools        []ToolDefinition  `json:"tools,omitempty"`
	ToolChoice   string            `json:"tool_choice,omitempty"`
	APIKey       string            `json:"-"`
	APIBase      string            `json:"-"`
	ExtraHeaders map[string]string `json:"-"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content          string            `json:"content"`
			ToolCalls        []ToolCallRequest `json:"tool_calls,omitempty"`
			ReasoningContent string            `json:"reasoning_content,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

func (p *LiteLLMProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, maxTokens int, temperature float64) (*LLMResponse, error) {
	if model == "" {
		model = p.defaultModel
	}

	// model = p.resolveModel(model)

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	p.applyModelOverrides(model, map[string]interface{}{
		"max_tokens":  maxTokens,
		"temperature": temperature,
	})

	if len(tools) > 0 {
		reqBody.Tools = tools
		reqBody.ToolChoice = "auto"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiBase := p.apiBase
	if apiBase == "" && p.gateway != nil {
		apiBase = p.gateway.DefaultAPIBase
	}
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(apiBase, "/"))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &LLMResponse{
			Content:      fmt.Sprintf("Error calling LLM: HTTP %d - %s", resp.StatusCode, string(body)),
			FinishReason: "error",
		}, nil
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return &LLMResponse{
			Content:      "No response from LLM",
			FinishReason: "stop",
		}, nil
	}

	choice := chatResp.Choices[0]
	response := &LLMResponse{
		Content:          choice.Message.Content,
		ToolCalls:        choice.Message.ToolCalls,
		FinishReason:     choice.FinishReason,
		ReasoningContent: choice.Message.ReasoningContent,
	}

	if chatResp.Usage.TotalTokens > 0 {
		response.Usage = map[string]int{
			"prompt_tokens":     chatResp.Usage.PromptTokens,
			"completion_tokens": chatResp.Usage.CompletionTokens,
			"total_tokens":      chatResp.Usage.TotalTokens,
		}
	}

	return response, nil
}

func (p *LiteLLMProvider) GetDefaultModel() string {
	return p.defaultModel
}
