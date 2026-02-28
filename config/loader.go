package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

type Loader struct {
	configPath string
	configName string
	viper      *viper.Viper
}

func NewLoader(configPath string) *Loader {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("json")

	homeDir, _ := os.UserHomeDir()
	defaultUserConfigDir := ""
	if strings.TrimSpace(homeDir) != "" {
		defaultUserConfigDir = filepath.Join(homeDir, ".nanobot")
	}

	if configPath != "" {
		// Support both: --config <file.json> OR --config <dir>
		if strings.HasSuffix(strings.ToLower(configPath), ".json") {
			v.SetConfigFile(configPath)
		} else {
			v.AddConfigPath(configPath)
			v.AddConfigPath(filepath.Join(configPath, "config"))
		}
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		if defaultUserConfigDir != "" {
			v.AddConfigPath(defaultUserConfigDir)
		}
		v.AddConfigPath("/etc/nanobot")
	}

	v.SetEnvPrefix("NANOBOT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Loader{
		configPath: configPath,
		configName: "config",
		viper:      v,
	}
}

func (l *Loader) defaultConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(homeDir) == "" {
		return filepath.Join(".", "config.json")
	}
	return filepath.Join(homeDir, ".nanobot", "config.json")
}

func (l *Loader) Load() (*Config, error) {
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return l.getDefaultConfig(), nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := l.viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func (l *Loader) getDefaultConfig() *Config {
	return &Config{
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:         ".",
				Provider:          "openai",
				MaxTokens:         4096,
				Temperature:       0.7,
				MaxToolIterations: 15,
			},
		},
		Channels: ChannelsConfig{
			WhatsApp: WhatsAppConfig{
				Enabled:   false,
				BridgeURL: "http://localhost:3000",
				AllowFrom: []string{},
			},
			Telegram: TelegramConfig{
				Enabled:   false,
				Token:     "",
				AllowFrom: []string{},
				Proxy:     "",
			},
			Discord: DiscordConfig{
				Enabled:    false,
				Token:      "",
				AllowFrom:  []string{},
				GatewayURL: "",
				Intents:    0,
			},
			Feishu: FeishuConfig{
				Enabled:           false,
				AppID:             "",
				AppSecret:         "",
				EncryptKey:        "",
				VerificationToken: "",
				AllowFrom:         []string{},
			},
			Mochat: MochatConfig{
				Enabled:                 false,
				BaseURL:                 "http://localhost:8080",
				SocketURL:               "ws://localhost:8080/socket.io",
				SocketPath:              "/socket.io",
				SocketDisableMsgpack:    false,
				SocketReconnectDelay:    1000,
				SocketMaxReconnectDelay: 30000,
				SocketConnectTimeout:    10000,
				RefreshInterval:         60000,
				WatchTimeout:            30000,
				WatchLimit:              100,
				RetryDelay:              1000,
				MaxRetryAttempts:        3,
				ClawToken:               "",
				AgentUserID:             "",
				Sessions:                []string{},
				Panels:                  []string{},
				AllowFrom:               []string{},
				Mention: MochatMentionConfig{
					RequireInGroups: true,
				},
				Groups:         map[string]MochatGroupRule{},
				ReplyDelayMode: "none",
				ReplyDelayMS:   0,
			},
			DingTalk: DingTalkConfig{
				Enabled:      false,
				ClientID:     "",
				ClientSecret: "",
				AllowFrom:    []string{},
			},
			Email: EmailConfig{
				Enabled:          false,
				ConsentGranted:   false,
				IMAPHost:         "",
				IMAPPort:         993,
				IMAPUsername:     "",
				IMAPPassword:     "",
				IMAPMailbox:      "INBOX",
				IMAPUseSSL:       true,
				SMTPHost:         "",
				SMTPPort:         587,
				SMTPUsername:     "",
				SMTPPassword:     "",
				SMTPUseTLS:       true,
				SMTPUseSSL:       false,
				FromAddress:      "",
				AutoReplyEnabled: false,
				PollInterval:     30,
				MarkSeen:         false,
				MaxBodyChars:     10000,
				SubjectPrefix:    "",
				AllowFrom:        []string{},
			},
			Slack: SlackConfig{
				Enabled:           false,
				Mode:              "webhook",
				WebhookPath:       "/slack/webhook",
				BotToken:          "",
				AppToken:          "",
				UserTokenReadOnly: false,
				GroupPolicy:       "allow_all",
				GroupAllowFrom:    []string{},
				DM: SlackDMConfig{
					Enabled:   false,
					Policy:    "allow_all",
					AllowFrom: []string{},
				},
			},
			QQ: QQConfig{
				Enabled:   false,
				AppID:     "",
				Secret:    "",
				AllowFrom: []string{},
			},
			Web: WebConfig{
				Enabled:   true,
				AllowFrom: []string{},
			},
		},
		Providers: map[string]ProviderConfig{
			"anthropic": {
				APIKey:  "",
				APIBase: "https://api.anthropic.com",
				ExtraHeaders: map[string]string{
					"anthropic-version": "2023-06-01",
				},
			},
			"openai": {
				APIKey:       "",
				APIBase:      "https://api.openai.com/v1",
				ExtraHeaders: map[string]string{},
			},
			"openrouter": {
				APIKey:       "",
				APIBase:      "https://openrouter.ai/api/v1",
				ExtraHeaders: map[string]string{},
			},
			"deepseek": {
				APIKey:       "",
				APIBase:      "https://api.deepseek.com",
				ExtraHeaders: map[string]string{},
			},
			"groq": {
				APIKey:       "",
				APIBase:      "https://api.groq.com/openai/v1",
				ExtraHeaders: map[string]string{},
			},
			"zhipu": {
				APIKey:       "",
				APIBase:      "https://open.bigmodel.cn/api/paas/v4",
				ExtraHeaders: map[string]string{},
			},
			"dashscope": {
				APIKey:       "",
				APIBase:      "https://dashscope.aliyuncs.com/compatible-mode/v1",
				ExtraHeaders: map[string]string{},
			},
			"vllm": {
				APIKey:       "",
				APIBase:      "http://localhost:8000/v1",
				ExtraHeaders: map[string]string{},
			},
			"gemini": {
				APIKey:       "",
				APIBase:      "https://generativelanguage.googleapis.com/v1beta",
				ExtraHeaders: map[string]string{},
			},
			"moonshot": {
				APIKey:       "",
				APIBase:      "https://api.moonshot.cn/v1",
				ExtraHeaders: map[string]string{},
			},
			"minimax": {
				APIKey:       "",
				APIBase:      "https://api.minimax.chat/v1",
				ExtraHeaders: map[string]string{},
			},
			"aihubmix": {
				APIKey:       "",
				APIBase:      "https://aihubmix.com/v1",
				ExtraHeaders: map[string]string{},
			},
		},
		Gateway: GatewayConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Tools: ToolsConfig{
			Web: WebToolsConfig{
				Search: WebSearchConfig{
					APIKey:     "",
					MaxResults: 5,
				},
			},
			Exec: ExecToolConfig{
				Timeout: 60,
			},
			RestrictWorkspace: false,
		},
	}
}

func (l *Loader) Save(cfg *Config) error {
	configFile := l.viper.ConfigFileUsed()
	if configFile == "" {
		// If user passed --config, treat it as either file path or directory.
		if l.configPath != "" {
			if strings.HasSuffix(strings.ToLower(l.configPath), ".json") {
				configFile = l.configPath
			} else {
				configFile = filepath.Join(l.configPath, "config.json")
			}
		} else {
			// Default: ~/.nanobot/config.json
			configFile = l.defaultConfigFilePath()
		}
	}

	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Use JSON encoding to preserve camelCase field names
	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling config to JSON: %w", err)
	}

	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

func (l *Loader) GetConfigPath() string {
	if used := l.viper.ConfigFileUsed(); used != "" {
		return used
	}
	if l.configPath != "" {
		if strings.HasSuffix(strings.ToLower(l.configPath), ".json") {
			return l.configPath
		}
		return filepath.Join(l.configPath, "config.json")
	}
	return l.defaultConfigFilePath()
}

func (l *Loader) GetConfigDir() string {
	configFile := l.viper.ConfigFileUsed()
	if configFile == "" {
		return l.configPath
	}
	return filepath.Dir(configFile)
}

func (l *Loader) GetDefaultConfig() *Config {
	return l.getDefaultConfig()
}

func setStructWithMapstructure(viper *viper.Viper, prefix string, value interface{}) {
	t := reflect.TypeOf(value)
	val := reflect.ValueOf(value)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		val = val.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get the mapstructure tag
		tag := field.Tag.Get("mapstructure")
		if tag == "" {
			// If no tag, use the field name converted to snake_case
			tag = toSnakeCase(field.Name)
		}

		// Skip empty tags
		if tag == "" {
			continue
		}

		// Build the key path
		key := tag
		if prefix != "" {
			key = prefix + "." + tag
		}

		// Check if the field is a struct
		if fieldValue.Kind() == reflect.Struct {
			// Recursively process nested structs
			setStructWithMapstructure(viper, key, fieldValue.Interface())
		} else if fieldValue.Kind() == reflect.Map || fieldValue.Kind() == reflect.Slice {
			// For maps and slices, set them directly
			viper.Set(key, fieldValue.Interface())
		} else {
			// For primitive types, set them directly
			viper.Set(key, fieldValue.Interface())
		}
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
