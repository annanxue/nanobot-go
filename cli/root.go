package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/nanobotgo/agent"
	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/channels"
	"github.com/nanobotgo/config"
	"github.com/nanobotgo/cron"
	"github.com/nanobotgo/heartbeat"
	"github.com/nanobotgo/providers"
	"github.com/nanobotgo/session"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	Version = "0.1.0"
	Logo    = "🐈"
)

var rootCmd = &cobra.Command{
	Use:   "nanobotgo",
	Short: fmt.Sprintf("%s nanobotgo - Personal AI Assistant", Logo),
	Long:  fmt.Sprintf("%s nanobotgo - Personal AI Assistant\n\nA lightweight AI assistant with tool support.", Logo),
	RunE:  runRoot,
}

var (
	verbose    bool
	configPath string
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func loadConfig() (*config.Config, error) {
	loader := config.NewLoader(configPath)
	cfg, err := loader.Load()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func setupLogging(verbose bool) {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

func makeProvider(cfg *config.Config) (providers.LLMProvider, error) {
	provider := cfg.Agents.Defaults.Provider
	providerCfg := getProviderConfig(cfg, provider)

	if providerCfg == nil {
		return nil, fmt.Errorf("no API key configured for provider")
	}

	return providers.NewOpenAIProvider( // litellm_provider对于deepseek的role:tool支持有问题，content会改成数组，deepseek不支持
		providerCfg.APIKey,
		providerCfg.APIBase,
		providerCfg.Model,
		providerCfg.ExtraHeaders,
		cfg.Agents.Defaults.Provider,
	), nil
}

func getProviderConfig(cfg *config.Config, providerName string) *config.ProviderConfig {
	if providerCfg, ok := cfg.Providers[providerName]; ok {
		return &providerCfg
	}
	openAIConfig, ok := cfg.Providers["openai"]
	if !ok {
		return &config.ProviderConfig{}
	}
	return &openAIConfig
}

func getProviderNameFromModel(model string) string {
	if len(model) == 0 {
		return "openai"
	}

	prefixes := map[string]string{
		"claude":            "anthropic",
		"gpt":               "openai",
		"deepseek-reasoner": "deepseek",
		"llama":             "groq",
		"glm":               "zhipu",
		// "qwen":        "dashscope",
		"gemini":      "gemini",
		"moonshot":    "moonshot",
		"minimax":     "minimax",
		"qwen3-vl:8b": "ollama",
	}

	for prefix, provider := range prefixes {
		if len(model) >= len(prefix) && model[:len(prefix)] == prefix {
			return provider
		}
	}

	return "openai"
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logrus.Info("Received interrupt signal, shutting down...")
		cancel()
	}()
}

func createWorkspaceTemplates(workspace string) error {
	templates := map[string]string{
		"AGENTS.md": `# Agent Instructions

You are a helpful AI assistant. Be concise, accurate, and friendly.

## Guidelines

- Always explain what you're doing before taking actions
- Ask for clarification when request is ambiguous
- Use tools to help accomplish tasks
- Remember important information in your memory files
`,
		"SOUL.md": `# Soul

I am nanobot, a lightweight AI assistant.

## Personality

- Helpful and friendly
- Concise and to the point
- Curious and eager to learn

## Values

- Accuracy over speed
- User privacy and safety
- Transparency in actions
`,
		"USER.md": `# User

Information about the user goes here.

## Preferences

- Communication style: (casual/formal)
- Timezone: (your timezone)
- Language: (your preferred language)
`,
	}

	for filename, content := range templates {
		filePath := filepath.Join(workspace, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				logrus.Warnf("Failed to create %s: %v", filename, err)
			} else {
				logrus.Infof("Created %s", filename)
			}
		}
	}

	memoryDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		return err
	}

	memoryFile := filepath.Join(memoryDir, "MEMORY.md")
	if _, err := os.Stat(memoryFile); os.IsNotExist(err) {
		memoryContent := `# Long-term Memory

This file stores important information that should persist across sessions.

## User Information

(Important facts about the user)

## Preferences

(User preferences learned over time)

## Important Notes

(Things to remember)
`
		if err := os.WriteFile(memoryFile, []byte(memoryContent), 0644); err != nil {
			return err
		}
		logrus.Info("Created memory/MEMORY.md")
	}

	skillsDir := filepath.Join(workspace, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	return nil
}

func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".nanobot"
	}
	return filepath.Join(homeDir, ".nanobot")
}

func getWorkspacePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, ".nanobot", "workspace")
}

func createAgentLoop(
	cfg *config.Config,
	bus *bus.MessageBus,
	provider providers.LLMProvider,
	sessionManager *session.SessionManager,
	cronService *cron.CronService,
) *agent.AgentLoop {
	return agent.NewAgentLoop(
		bus,
		provider,
		cfg.Agents.Defaults.Workspace,
		provider.GetDefaultModel(),
		cfg.Agents.Defaults.MaxToolIterations,
		cfg.Tools.Web.Search.APIKey,
		&agent.ExecToolConfig{Timeout: cfg.Tools.Exec.Timeout},
		cronService,
		cfg.Tools.RestrictWorkspace,
		sessionManager,
	)
}

func createChannelManager(cfg *config.Config, bus *bus.MessageBus, sessionManager *session.SessionManager) (*channels.ChannelManager, error) {
	cm := channels.NewChannelManager(cfg, bus, sessionManager)
	if err := cm.InitChannels(); err != nil {
		return nil, err
	}
	return cm, nil
}

func createCronService(cfg *config.Config) (*cron.CronService, error) {
	dataDir := getDataDir()
	cronStorePath := filepath.Join(dataDir, "cron", "jobs.json")
	return cron.NewCronService(cronStorePath), nil
}

func createHeartbeatService(cfg *config.Config, onHeartbeat func(string) (string, error)) *heartbeat.HeartbeatService {
	return heartbeat.NewHeartbeatService(
		cfg.Agents.Defaults.Workspace,
		onHeartbeat,
		30*60,
		true,
	)
}

func createSessionManager(cfg *config.Config) *session.SessionManager {
	return session.NewSessionManager(cfg.Agents.Defaults.Workspace)
}
