package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nanobotgo/agent"
	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	agentMessage  string
	agentSession  string
	agentMarkdown bool
	agentLogs     bool
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Interact with the agent directly",
	RunE:  runAgent,
}

func init() {
	agentCmd.Flags().StringVarP(&agentMessage, "message", "m", "", "Message to send to the agent")
	agentCmd.Flags().StringVarP(&agentSession, "session", "s", "cli:default", "Session ID")
	agentCmd.Flags().BoolVar(&agentMarkdown, "markdown", true, "Render assistant output as Markdown")
	agentCmd.Flags().BoolVar(&agentLogs, "logs", false, "Show nanobot runtime logs during chat")

	rootCmd.AddCommand(agentCmd)
}

func runAgent(cmd *cobra.Command, args []string) error {
	setupLogging(verbose)

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	bus := bus.NewMessageBus()
	provider, err := makeProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	if agentLogs {
		utils.Log.SetLevel(logrus.DebugLevel)
	} else {
		utils.Log.SetLevel(logrus.ErrorLevel)
	}

	sessionManager := createSessionManager(cfg)

	agentLoop := agent.NewAgentLoop(
		"default",
		bus,
		provider,
		cfg.Agents.Defaults.Workspace,
		provider.GetDefaultModel(),
		cfg.Agents.Defaults.MaxToolIterations,
		cfg.Tools.Web.Search.APIKey,
		&agent.ExecToolConfig{Timeout: cfg.Tools.Exec.Timeout},
		nil,
		cfg.Tools.RestrictWorkspace,
		sessionManager,
	)

	if agentMessage != "" {
		return runSingleMessage(agentLoop, agentMessage, agentSession)
	}

	return runInteractive(agentLoop, agentSession)
}

func runSingleMessage(agentLoop *agent.AgentLoop, message, sessionKey string) error {
	fmt.Printf("%s nanobot is thinking...\n", Logo)

	response, err := agentLoop.ProcessDirect(context.Background(), message, sessionKey, "cli", "direct", nil)
	if err != nil {
		return fmt.Errorf("agent error: %w", err)
	}

	printAgentResponse(response, agentMarkdown)
	return nil
}

func runInteractive(agentLoop *agent.AgentLoop, sessionKey string) error {
	fmt.Printf("%s Interactive mode (type 'exit' or Ctrl+C to quit)\n\n", Logo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandler(cancel)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if isExitCommand(input) {
			fmt.Println("\nGoodbye!")
			break
		}

		fmt.Printf("%s nanobot is thinking...\n", Logo)

		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey, "cli", "direct", nil)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		printAgentResponse(response, agentMarkdown)
	}

	return nil
}

func printAgentResponse(response string, renderMarkdown bool) {
	fmt.Println()
	fmt.Printf("%s nanobot\n", Logo)
	fmt.Println(response)
	fmt.Println()
}

func isExitCommand(command string) bool {
	lower := strings.ToLower(command)
	return lower == "exit" || lower == "quit" || lower == "/exit" || lower == "/quit" || lower == ":q"
}
