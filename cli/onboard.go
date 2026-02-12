package cli

import (
	"fmt"
	"os"

	"github.com/nanobotgo/config"
	"github.com/spf13/cobra"
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Initialize nanobot configuration and workspace",
	RunE:  runOnboard,
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}

func runOnboard(cmd *cobra.Command, args []string) error {
	loader := config.NewLoader(configPath)
	cfgPath := loader.GetConfigPath()

	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("Config already exists at %s\n", cfgPath)
		fmt.Print("Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return nil
		}
	}

	defaultCfg := loader.GetDefaultConfig()
	if err := loader.Save(defaultCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("✓ Created config at %s\n", cfgPath)

	workspace := getWorkspacePath()
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}
	fmt.Printf("✓ Created workspace at %s\n", workspace)

	if err := createWorkspaceTemplates(workspace); err != nil {
		return fmt.Errorf("failed to create workspace templates: %w", err)
	}

	fmt.Printf("\n%s nanobot is ready!\n\n", Logo)
	fmt.Println("Next steps:")
	fmt.Println("  1. Add your API key to ~/.nanobot/config.json")
	fmt.Println("     Get one at: https://openrouter.ai/keys")
	fmt.Println("  2. Chat: nanobotgo agent -m \"Hello!\"")
	fmt.Println("\nWant Telegram/WhatsApp? See: https://github.com/HKUDS/nanobot#-chat-apps")

	return nil
}
