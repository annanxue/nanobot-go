package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nanobotgo/agent"
	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/channels"
	"github.com/nanobotgo/config"
	"github.com/nanobotgo/cron"
	"github.com/nanobotgo/utils"
	"github.com/nanobotgo/webui"
	"github.com/spf13/cobra"
)

var (
	gatewayPort    int
	gatewayVerbose bool
)

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start nanobot gateway",
	RunE:  runGateway,
}

func init() {
	gatewayCmd.Flags().IntVarP(&gatewayPort, "port", "p", 18790, "Gateway port")
	gatewayCmd.Flags().BoolVarP(&gatewayVerbose, "verbose", "v", false, "Verbose output")

	rootCmd.AddCommand(gatewayCmd)
}

func runGateway(cmd *cobra.Command, args []string) error {
	setupLogging(gatewayVerbose)

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	utils.Log.Infof("Starting nanobot gateway on port %d...", gatewayPort)

	messageBus := bus.NewMessageBus()

	sessionManager := createSessionManager(cfg)

	cronService, err := createCronService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create cron service: %w", err)
	}

	agentLoops := make(map[string]*agent.AgentLoop)

	defaultProvider, err := makeProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to create default provider: %w", err)
	}
	defaultAgentLoop := createAgentLoop("default", cfg, messageBus, defaultProvider, sessionManager, cronService)
	agentLoops["default"] = defaultAgentLoop

	for _, agentCfg := range cfg.Agents.Agents {
		agentLoop, err := createAgentLoopWithConfig(agentCfg, cfg, messageBus, sessionManager, cronService)
		if err != nil {
			utils.Log.Warnf("Failed to create agent %s: %v, using default", agentCfg.Name, err)
			continue
		}
		agentLoops[agentCfg.Name] = agentLoop
		utils.Log.Infof("Agent [%s] started with provider=%s, model=%s, workspace=%s",
			agentCfg.Name, agentCfg.Provider, agentLoop.GetModel(), agentLoop.GetWorkspace())
	}

	agentDispatcher := NewAgentDispatcher(agentLoops, defaultAgentLoop, messageBus)

	cronService.SetOnJob(func(job *cron.CronJob) (string, error) {
		// response, err := agentLoop.ProcessDirect(
		// 	context.Background(),
		// 	job.Payload.Message,
		// 	fmt.Sprintf("cron:%s", job.ID),
		// 	job.Payload.Channel,
		// 	job.Payload.To,
		// 	nil,
		// )
		// if err != nil {
		// 	utils.Log.Errorf("Cron job error: %v", err)
		// 	return "", err
		// }

		if job.Payload.Deliver && job.Payload.To != "" {
			messageBus.PublishOutbound(&bus.OutboundMessage{
				Channel: job.Payload.Channel,
				ChatID:  job.Payload.To,
				Content: job.Payload.Message,
			})
		}
		return job.Payload.Message, nil
	})

	heartbeatService := createHeartbeatService(cfg, func(prompt string) (string, error) {
		response, err := defaultAgentLoop.ProcessDirect(
			context.Background(),
			prompt,
			"heartbeat",
			"cli",
			"direct",
			nil,
		)
		if err != nil {
			utils.Log.Errorf("Heartbeat error: %v", err)
			return "", err
		}
		return response, nil
	})

	channelManager, err := createChannelManager(cfg, messageBus, sessionManager)
	if err != nil {
		return fmt.Errorf("failed to create channel manager: %w", err)
	}

	if len(channelManager.EnabledChannels()) > 0 {
		utils.Log.Infof("Channels enabled: %s", stringsJoin(channelManager.EnabledChannels(), ", "))
	} else {
		utils.Log.Warn("No channels enabled")
	}

	cronStatus := cronService.Status()
	jobsCount := 0
	if count, ok := cronStatus["jobs"].(int); ok {
		jobsCount = count
	}
	if jobsCount > 0 {
		utils.Log.Infof("Cron: %d scheduled jobs", jobsCount)
	}

	utils.Log.Info("Heartbeat: every 30m")

	go func() {
		loader := config.NewLoader(configPath)
		cfg, err := loader.Load()
		if err != nil {
			utils.Log.Warnf("WebUI: Failed to load config: %v", err)
			return
		}
		skillsLoader := agent.NewSkillsLoader(cfg.Agents.Defaults.Workspace)
		webUIServer := webui.NewServer(cfg, loader.GetConfigPath(), loader, cronService, sessionManager, skillsLoader, messageBus, channelManager.GetChannel("web").(*channels.WebChannel), ":18080", agentDispatcher.GetAgentLoops())
		if err := webUIServer.Start(); err != nil {
			utils.Log.Warnf("WebUI server error: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		utils.Log.Info("Shutting down...")
		heartbeatService.Stop()
		cronService.Stop()
		agentDispatcher.StopAll()
		if err := channelManager.StopAll(ctx); err != nil {
			utils.Log.Errorf("Error stopping channels: %v", err)
		}
		cancel()
	}()

	if err := cronService.Start(); err != nil {
		return fmt.Errorf("failed to start cron service: %w", err)
	}

	if err := heartbeatService.Start(); err != nil {
		return fmt.Errorf("failed to start heartbeat service: %w", err)
	}

	for name, agentLoop := range agentDispatcher.GetAgentLoops() {
		name := name
		agentLoop := agentLoop
		go func() {
			if err := agentLoop.Run(ctx); err != nil {
				utils.Log.Errorf("Agent loop [%s] error: %v", name, err)
			}
		}()
	}

	agentDispatcher.StartConsuming(ctx)

	if err := channelManager.StartAll(ctx); err != nil {
		return fmt.Errorf("failed to start channels: %w", err)
	}

	<-ctx.Done()
	return nil
}

func stringsJoin(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for i := 1; i < len(items); i++ {
		result += sep + items[i]
	}
	return result
}
