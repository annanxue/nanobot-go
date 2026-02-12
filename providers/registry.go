package providers

import "strings"

type ProviderSpec struct {
	Name                string          `json:"name"`
	Keywords            []string        `json:"keywords"`
	EnvKey              string          `json:"env_key"`
	DisplayName         string          `json:"display_name"`
	LiteLLMPrefix       string          `json:"litellm_prefix"`
	SkipPrefixes        []string        `json:"skip_prefixes"`
	EnvExtras           []EnvExtra      `json:"env_extras"`
	IsGateway           bool            `json:"is_gateway"`
	IsLocal             bool            `json:"is_local"`
	DetectByKeyPrefix   string          `json:"detect_by_key_prefix"`
	DetectByBaseKeyword string          `json:"detect_by_base_keyword"`
	DefaultAPIBase      string          `json:"default_api_base"`
	StripModelPrefix    bool            `json:"strip_model_prefix"`
	ModelOverrides      []ModelOverride `json:"model_overrides"`
}

type EnvExtra struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ModelOverride struct {
	Pattern   string                 `json:"pattern"`
	Overrides map[string]interface{} `json:"overrides"`
}

var PROVIDERS = []ProviderSpec{
	{
		Name:                "openrouter",
		Keywords:            []string{"openrouter"},
		EnvKey:              "OPENROUTER_API_KEY",
		DisplayName:         "OpenRouter",
		LiteLLMPrefix:       "openrouter",
		SkipPrefixes:        []string{},
		EnvExtras:           []EnvExtra{},
		IsGateway:           true,
		IsLocal:             false,
		DetectByKeyPrefix:   "sk-or-",
		DetectByBaseKeyword: "openrouter",
		DefaultAPIBase:      "https://openrouter.ai/api/v1",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "aihubmix",
		Keywords:            []string{"aihubmix"},
		EnvKey:              "OPENAI_API_KEY",
		DisplayName:         "AiHubMix",
		LiteLLMPrefix:       "openai",
		SkipPrefixes:        []string{},
		EnvExtras:           []EnvExtra{},
		IsGateway:           true,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "aihubmix",
		DefaultAPIBase:      "https://aihubmix.com/v1",
		StripModelPrefix:    true,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "anthropic",
		Keywords:            []string{"anthropic", "claude"},
		EnvKey:              "ANTHROPIC_API_KEY",
		DisplayName:         "Anthropic",
		LiteLLMPrefix:       "",
		SkipPrefixes:        []string{},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "openai",
		Keywords:            []string{"openai", "gpt"},
		EnvKey:              "OPENAI_API_KEY",
		DisplayName:         "OpenAI",
		LiteLLMPrefix:       "",
		SkipPrefixes:        []string{},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "deepseek",
		Keywords:            []string{"deepseek"},
		EnvKey:              "DEEPSEEK_API_KEY",
		DisplayName:         "DeepSeek",
		LiteLLMPrefix:       "deepseek",
		SkipPrefixes:        []string{"deepseek/"},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "gemini",
		Keywords:            []string{"gemini"},
		EnvKey:              "GEMINI_API_KEY",
		DisplayName:         "Gemini",
		LiteLLMPrefix:       "gemini",
		SkipPrefixes:        []string{"gemini/"},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "zhipu",
		Keywords:            []string{"zhipu", "glm", "zai"},
		EnvKey:              "ZAI_API_KEY",
		DisplayName:         "Zhipu AI",
		LiteLLMPrefix:       "zai",
		SkipPrefixes:        []string{"zhipu/", "zai/", "openrouter/", "hosted_vllm/"},
		EnvExtras:           []EnvExtra{{Name: "ZHIPUAI_API_KEY", Value: "{api_key}"}},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "dashscope",
		Keywords:            []string{"qwen", "dashscope"},
		EnvKey:              "DASHSCOPE_API_KEY",
		DisplayName:         "DashScope",
		LiteLLMPrefix:       "dashscope",
		SkipPrefixes:        []string{"dashscope/", "openrouter/"},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "moonshot",
		Keywords:            []string{"moonshot", "kimi"},
		EnvKey:              "MOONSHOT_API_KEY",
		DisplayName:         "Moonshot",
		LiteLLMPrefix:       "moonshot",
		SkipPrefixes:        []string{"moonshot/", "openrouter/"},
		EnvExtras:           []EnvExtra{{Name: "MOONSHOT_API_BASE", Value: "{api_base}"}},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "https://api.moonshot.ai/v1",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{{Pattern: "kimi-k2.5", Overrides: map[string]interface{}{"temperature": 1.0}}},
	},
	{
		Name:                "minimax",
		Keywords:            []string{"minimax"},
		EnvKey:              "MINIMAX_API_KEY",
		DisplayName:         "MiniMax",
		LiteLLMPrefix:       "minimax",
		SkipPrefixes:        []string{"minimax/", "openrouter/"},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "https://api.minimax.io/v1",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "vllm",
		Keywords:            []string{"vllm"},
		EnvKey:              "HOSTED_VLLM_API_KEY",
		DisplayName:         "vLLM/Local",
		LiteLLMPrefix:       "hosted_vllm",
		SkipPrefixes:        []string{},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             true,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
	{
		Name:                "groq",
		Keywords:            []string{"groq"},
		EnvKey:              "GROQ_API_KEY",
		DisplayName:         "Groq",
		LiteLLMPrefix:       "groq",
		SkipPrefixes:        []string{"groq/"},
		EnvExtras:           []EnvExtra{},
		IsGateway:           false,
		IsLocal:             false,
		DetectByKeyPrefix:   "",
		DetectByBaseKeyword: "",
		DefaultAPIBase:      "",
		StripModelPrefix:    false,
		ModelOverrides:      []ModelOverride{},
	},
}

func FindByModel(model string) *ProviderSpec {
	modelLower := strings.ToLower(model)
	for _, spec := range PROVIDERS {
		if spec.IsGateway || spec.IsLocal {
			continue
		}
		for _, kw := range spec.Keywords {
			if strings.Contains(modelLower, kw) {
				return &spec
			}
		}
	}
	return nil
}

func FindGateway(providerName, apiKey, apiBase string) *ProviderSpec {
	if providerName != "" {
		spec := FindByName(providerName)
		if spec != nil && (spec.IsGateway || spec.IsLocal) {
			return spec
		}
	}

	for _, spec := range PROVIDERS {
		if spec.DetectByKeyPrefix != "" && apiKey != "" && strings.HasPrefix(apiKey, spec.DetectByKeyPrefix) {
			return &spec
		}
		if spec.DetectByBaseKeyword != "" && apiBase != "" && strings.Contains(apiBase, spec.DetectByBaseKeyword) {
			return &spec
		}
	}
	return nil
}

func FindByName(name string) *ProviderSpec {
	for _, spec := range PROVIDERS {
		if spec.Name == name {
			return &spec
		}
	}
	return nil
}
