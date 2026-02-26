package config

type WhatsAppConfig struct {
	Enabled   bool     `json:"enabled" mapstructure:"enabled"`
	BridgeURL string   `json:"bridgeUrl" mapstructure:"bridgeUrl"`
	AllowFrom []string `json:"allowFrom" mapstructure:"allowFrom"`
}

type TelegramConfig struct {
	Enabled   bool     `json:"enabled" mapstructure:"enabled"`
	Token     string   `json:"token" mapstructure:"token"`
	AllowFrom []string `json:"allowFrom" mapstructure:"allowFrom"`
	Proxy     string   `json:"proxy" mapstructure:"proxy"`
}

type FeishuConfig struct {
	Enabled           bool     `json:"enabled" mapstructure:"enabled"`
	AppID             string   `json:"appId" mapstructure:"appId"`
	AppSecret         string   `json:"appSecret" mapstructure:"appSecret"`
	EncryptKey        string   `json:"encryptKey" mapstructure:"encryptKey"`
	VerificationToken string   `json:"verificationToken" mapstructure:"verificationToken"`
	AllowFrom         []string `json:"allowFrom" mapstructure:"allowFrom"`
}

type DingTalkConfig struct {
	Enabled      bool     `json:"enabled" mapstructure:"enabled"`
	ClientID     string   `json:"clientId" mapstructure:"clientId"`
	ClientSecret string   `json:"clientSecret" mapstructure:"clientSecret"`
	AllowFrom    []string `json:"allowFrom" mapstructure:"allowFrom"`
}

type DiscordConfig struct {
	Enabled    bool     `json:"enabled" mapstructure:"enabled"`
	Token      string   `json:"token" mapstructure:"token"`
	AllowFrom  []string `json:"allowFrom" mapstructure:"allowFrom"`
	GatewayURL string   `json:"gatewayUrl" mapstructure:"gatewayUrl"`
	Intents    int      `json:"intents" mapstructure:"intents"`
}

type EmailConfig struct {
	Enabled          bool     `json:"enabled" mapstructure:"enabled"`
	ConsentGranted   bool     `json:"consentGranted" mapstructure:"consentGranted"`
	IMAPHost         string   `json:"imapHost" mapstructure:"imapHost"`
	IMAPPort         int      `json:"imapPort" mapstructure:"imapPort"`
	IMAPUsername     string   `json:"imapUsername" mapstructure:"imapUsername"`
	IMAPPassword     string   `json:"imapPassword" mapstructure:"imapPassword"`
	IMAPMailbox      string   `json:"imapMailbox" mapstructure:"imapMailbox"`
	IMAPUseSSL       bool     `json:"imapUseSsl" mapstructure:"imapUseSsl"`
	SMTPHost         string   `json:"smtpHost" mapstructure:"smtpHost"`
	SMTPPort         int      `json:"smtpPort" mapstructure:"smtpPort"`
	SMTPUsername     string   `json:"smtpUsername" mapstructure:"smtpUsername"`
	SMTPPassword     string   `json:"smtpPassword" mapstructure:"smtpPassword"`
	SMTPUseTLS       bool     `json:"smtpUseTls" mapstructure:"smtpUseTls"`
	SMTPUseSSL       bool     `json:"smtpUseSsl" mapstructure:"smtpUseSsl"`
	FromAddress      string   `json:"fromAddress" mapstructure:"fromAddress"`
	AutoReplyEnabled bool     `json:"autoReplyEnabled" mapstructure:"autoReplyEnabled"`
	PollInterval     int      `json:"pollIntervalSeconds" mapstructure:"pollIntervalSeconds"`
	MarkSeen         bool     `json:"markSeen" mapstructure:"markSeen"`
	MaxBodyChars     int      `json:"maxBodyChars" mapstructure:"maxBodyChars"`
	SubjectPrefix    string   `json:"subjectPrefix" mapstructure:"subjectPrefix"`
	AllowFrom        []string `json:"allowFrom" mapstructure:"allowFrom"`
}

type MochatMentionConfig struct {
	RequireInGroups bool `json:"requireInGroups" mapstructure:"requireInGroups"`
}

type MochatGroupRule struct {
	RequireMention bool `json:"requireMention" mapstructure:"requireMention"`
}

type MochatConfig struct {
	Enabled                 bool                       `json:"enabled" mapstructure:"enabled"`
	BaseURL                 string                     `json:"baseUrl" mapstructure:"baseUrl"`
	SocketURL               string                     `json:"socketUrl" mapstructure:"socketUrl"`
	SocketPath              string                     `json:"socketPath" mapstructure:"socketPath"`
	SocketDisableMsgpack    bool                       `json:"socketDisableMsgpack" mapstructure:"socketDisableMsgpack"`
	SocketReconnectDelay    int                        `json:"socketReconnectDelayMs" mapstructure:"socketReconnectDelayMs"`
	SocketMaxReconnectDelay int                        `json:"socketMaxReconnectDelayMs" mapstructure:"socketMaxReconnectDelayMs"`
	SocketConnectTimeout    int                        `json:"socketConnectTimeoutMs" mapstructure:"socketConnectTimeoutMs"`
	RefreshInterval         int                        `json:"refreshIntervalMs" mapstructure:"refreshIntervalMs"`
	WatchTimeout            int                        `json:"watchTimeoutMs" mapstructure:"watchTimeoutMs"`
	WatchLimit              int                        `json:"watchLimit" mapstructure:"watchLimit"`
	RetryDelay              int                        `json:"retryDelayMs" mapstructure:"retryDelayMs"`
	MaxRetryAttempts        int                        `json:"maxRetryAttempts" mapstructure:"maxRetryAttempts"`
	ClawToken               string                     `json:"clawToken" mapstructure:"clawToken"`
	AgentUserID             string                     `json:"agentUserId" mapstructure:"agentUserId"`
	Sessions                []string                   `json:"sessions" mapstructure:"sessions"`
	Panels                  []string                   `json:"panels" mapstructure:"panels"`
	AllowFrom               []string                   `json:"allowFrom" mapstructure:"allowFrom"`
	Mention                 MochatMentionConfig        `json:"mention" mapstructure:"mention"`
	Groups                  map[string]MochatGroupRule `json:"groups" mapstructure:"groups"`
	ReplyDelayMode          string                     `json:"replyDelayMode" mapstructure:"replyDelayMode"`
	ReplyDelayMS            int                        `json:"replyDelayMs" mapstructure:"replyDelayMs"`
}

type SlackDMConfig struct {
	Enabled   bool     `json:"enabled" mapstructure:"enabled"`
	Policy    string   `json:"policy" mapstructure:"policy"`
	AllowFrom []string `json:"allowFrom" mapstructure:"allowFrom"`
}

type SlackConfig struct {
	Enabled           bool          `json:"enabled" mapstructure:"enabled"`
	Mode              string        `json:"mode" mapstructure:"mode"`
	WebhookPath       string        `json:"webhookPath" mapstructure:"webhookPath"`
	BotToken          string        `json:"botToken" mapstructure:"botToken"`
	AppToken          string        `json:"appToken" mapstructure:"appToken"`
	UserTokenReadOnly bool          `json:"userTokenReadOnly" mapstructure:"userTokenReadOnly"`
	GroupPolicy       string        `json:"groupPolicy" mapstructure:"groupPolicy"`
	GroupAllowFrom    []string      `json:"groupAllowFrom" mapstructure:"groupAllowFrom"`
	DM                SlackDMConfig `json:"dm" mapstructure:"dm"`
}

type QQConfig struct {
	Enabled    bool     `json:"enabled" mapstructure:"enabled"`
	AppID      string   `json:"appId" mapstructure:"appId"`
	Secret     string   `json:"secret" mapstructure:"secret"`
	AllowFrom  []string `json:"allowFrom" mapstructure:"allowFrom"`
	WebhookURL string   `json:"webhookUrl" mapstructure:"webhookUrl"`
	SendURL    string   `json:"sendUrl" mapstructure:"sendUrl"`
}

type ChannelsConfig struct {
	WhatsApp WhatsAppConfig `json:"whatsapp" mapstructure:"whatsapp"`
	Telegram TelegramConfig `json:"telegram" mapstructure:"telegram"`
	Discord  DiscordConfig  `json:"discord" mapstructure:"discord"`
	Feishu   FeishuConfig   `json:"feishu" mapstructure:"feishu"`
	Mochat   MochatConfig   `json:"mochat" mapstructure:"mochat"`
	DingTalk DingTalkConfig `json:"dingtalk" mapstructure:"dingtalk"`
	Email    EmailConfig    `json:"email" mapstructure:"email"`
	Slack    SlackConfig    `json:"slack" mapstructure:"slack"`
	QQ       QQConfig       `json:"qq" mapstructure:"qq"`
}

type AgentDefaults struct {
	Workspace         string  `json:"workspace" mapstructure:"workspace"`
	Provider          string  `json:"provider" mapstructure:"provider"`
	MaxTokens         int     `json:"maxTokens" mapstructure:"maxTokens"`
	Temperature       float64 `json:"temperature" mapstructure:"temperature"`
	MaxToolIterations int     `json:"maxToolIterations" mapstructure:"maxToolIterations"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults" mapstructure:"defaults"`
}

type ProviderConfig struct {
	Model        string            `json:"model" mapstructure:"model"`
	APIKey       string            `json:"apiKey" mapstructure:"apiKey"`
	APIBase      string            `json:"apiBase" mapstructure:"apiBase"`
	ExtraHeaders map[string]string `json:"extraHeaders" mapstructure:"extraHeaders"`
}

type GatewayConfig struct {
	Host string `json:"host" mapstructure:"host"`
	Port int    `json:"port" mapstructure:"port"`
}

type WebSearchConfig struct {
	APIKey     string `json:"apiKey" mapstructure:"apiKey"`
	MaxResults int    `json:"maxResults" mapstructure:"maxResults"`
}

type WebToolsConfig struct {
	Search WebSearchConfig `json:"search" mapstructure:"search"`
}

type ExecToolConfig struct {
	Timeout int `json:"timeout" mapstructure:"timeout"`
}

type ToolsConfig struct {
	Web               WebToolsConfig `json:"web" mapstructure:"web"`
	Exec              ExecToolConfig `json:"exec" mapstructure:"exec"`
	RestrictWorkspace bool           `json:"restrictToWorkspace" mapstructure:"restrictToWorkspace"`
}

type Config struct {
	Agents    AgentsConfig              `json:"agents" mapstructure:"agents"`
	Channels  ChannelsConfig            `json:"channels" mapstructure:"channels"`
	Providers map[string]ProviderConfig `json:"providers" mapstructure:"providers"`
	Gateway   GatewayConfig             `json:"gateway" mapstructure:"gateway"`
	Tools     ToolsConfig               `json:"tools" mapstructure:"tools"`
}
