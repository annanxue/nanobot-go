package cli

import (
	"fmt"

	"github.com/nanobotgo/config"
	"github.com/spf13/cobra"
)

var channelsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show channel status",
	RunE:  runChannelsStatus,
}

var channelsApp = &cobra.Command{
	Use:   "channels",
	Short: "Manage channels",
}

func init() {
	channelsApp.AddCommand(channelsStatusCmd)
	rootCmd.AddCommand(channelsApp)
}

func runChannelsStatus(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("Channel Status")
	fmt.Println("────────────────────────────────────────────────────────────────────────")

	printChannelStatus("WhatsApp", cfg.Channels.WhatsApp.Enabled, cfg.Channels.WhatsApp.BridgeURL)
	printChannelStatus("Discord", cfg.Channels.Discord.Enabled, cfg.Channels.Discord.GatewayURL)
	printFeishuStatus(cfg.Channels.Feishu)
	printMochatStatus(cfg.Channels.Mochat)
	printTelegramStatus(cfg.Channels.Telegram)
	printSlackStatus(cfg.Channels.Slack)
	printQQStatus(cfg.Channels.QQ)
	printDingTalkStatus(cfg.Channels.DingTalk)
	printEmailStatus(cfg.Channels.Email)

	return nil
}

func printChannelStatus(name string, enabled bool, url string) {
	status := "✗"
	if enabled {
		status = "✓"
	}
	fmt.Printf("%-12s %-6s %s\n", name, status, url)
}

func printFeishuStatus(feishu config.FeishuConfig) {
	status := "✗"
	if feishu.Enabled {
		status = "✓"
	}
	configInfo := "not configured"
	if feishu.AppID != "" {
		configInfo = fmt.Sprintf("app_id: %s...", feishu.AppID[:min(10, len(feishu.AppID))])
	}
	fmt.Printf("%-12s %-6s %s\n", "Feishu", status, configInfo)
}

func printMochatStatus(mochat config.MochatConfig) {
	status := "✗"
	if mochat.Enabled {
		status = "✓"
	}
	configInfo := mochat.BaseURL
	if configInfo == "" {
		configInfo = "not configured"
	}
	fmt.Printf("%-12s %-6s %s\n", "Mochat", status, configInfo)
}

func printTelegramStatus(telegram config.TelegramConfig) {
	status := "✗"
	if telegram.Enabled {
		status = "✓"
	}
	configInfo := "not configured"
	if telegram.Token != "" {
		configInfo = fmt.Sprintf("token: %s...", telegram.Token[:min(10, len(telegram.Token))])
	}
	fmt.Printf("%-12s %-6s %s\n", "Telegram", status, configInfo)
}

func printSlackStatus(slack config.SlackConfig) {
	status := "✗"
	if slack.Enabled {
		status = "✓"
	}
	configInfo := "not configured"
	if slack.AppToken != "" && slack.BotToken != "" {
		configInfo = "socket"
	}
	fmt.Printf("%-12s %-6s %s\n", "Slack", status, configInfo)
}

func printQQStatus(qq config.QQConfig) {
	status := "✗"
	if qq.Enabled {
		status = "✓"
	}
	configInfo := "not configured"
	if qq.AppID != "" {
		configInfo = fmt.Sprintf("app_id: %s...", qq.AppID[:min(10, len(qq.AppID))])
	}
	fmt.Printf("%-12s %-6s %s\n", "QQ", status, configInfo)
}

func printDingTalkStatus(dingtalk config.DingTalkConfig) {
	status := "✗"
	if dingtalk.Enabled {
		status = "✓"
	}
	configInfo := "not configured"
	if dingtalk.ClientID != "" {
		configInfo = fmt.Sprintf("client_id: %s...", dingtalk.ClientID[:min(10, len(dingtalk.ClientID))])
	}
	fmt.Printf("%-12s %-6s %s\n", "DingTalk", status, configInfo)
}

func printEmailStatus(email config.EmailConfig) {
	status := "✗"
	if email.Enabled {
		status = "✓"
	}
	configInfo := "not configured"
	if email.IMAPHost != "" {
		configInfo = email.IMAPHost
	}
	fmt.Printf("%-12s %-6s %s\n", "Email", status, configInfo)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
