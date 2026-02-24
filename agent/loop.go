package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/cron"
	"github.com/nanobotgo/providers"
	"github.com/nanobotgo/session"
	"github.com/nanobotgo/tools"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
)

type AgentLoop struct {
	bus               *bus.MessageBus
	provider          providers.LLMProvider
	workspace         string
	model             string
	maxIterations     int
	braveAPIKey       string
	execConfig        *ExecToolConfig
	cronService       *cron.CronService
	restrictWorkspace bool
	sessionManager    *session.SessionManager
	context           *ContextBuilder
	toolsRegistry     *tools.ToolRegistry
	subagents         *SubagentManager
	running           bool
	mu                sync.RWMutex
}

type ExecToolConfig struct {
	Timeout int `json:"timeout"`
}

func NewAgentLoop(
	bus *bus.MessageBus,
	provider providers.LLMProvider,
	workspace string,
	model string,
	maxIterations int,
	braveAPIKey string,
	execConfig *ExecToolConfig,
	cronService *cron.CronService,
	restrictWorkspace bool,
	sessionManager *session.SessionManager,
) *AgentLoop {
	if execConfig == nil {
		execConfig = &ExecToolConfig{Timeout: 60}
	}

	al := &AgentLoop{
		bus:               bus,
		provider:          provider,
		workspace:         workspace,
		model:             model,
		maxIterations:     maxIterations,
		braveAPIKey:       braveAPIKey,
		execConfig:        execConfig,
		cronService:       cronService,
		restrictWorkspace: restrictWorkspace,
		sessionManager:    sessionManager,
		context:           NewContextBuilder(workspace),
		toolsRegistry:     tools.NewToolRegistry(),
	}

	al.subagents = NewSubagentManager(
		provider,
		workspace,
		bus,
		model,
		braveAPIKey,
		execConfig,
		restrictWorkspace,
	)

	al.registerDefaultTools()
	return al
}

func (al *AgentLoop) registerDefaultTools() {
	allowedDir := ""
	if al.restrictWorkspace {
		allowedDir = al.workspace
	}

	al.toolsRegistry.Register(tools.NewReadFileTool(allowedDir))
	al.toolsRegistry.Register(tools.NewWriteFileTool(allowedDir))
	al.toolsRegistry.Register(tools.NewEditFileTool(allowedDir))
	al.toolsRegistry.Register(tools.NewListDirTool(allowedDir))

	al.toolsRegistry.Register(tools.NewExecTool(
		al.execConfig.Timeout,
		al.workspace,
		nil,
		nil,
		al.restrictWorkspace,
	))

	al.toolsRegistry.Register(tools.NewWebSearchTool(al.braveAPIKey, 5))
	al.toolsRegistry.Register(tools.NewWebFetchTool(50000))

	messageTool := tools.NewMessageTool(func(msg *bus.OutboundMessage) error {
		al.bus.PublishOutbound(msg)
		return nil
	}, "", "")
	al.toolsRegistry.Register(messageTool)

	spawnTool := NewSpawnTool(al.subagents)
	al.toolsRegistry.Register(spawnTool)

	if al.cronService != nil {
		cronTool := NewCronTool(al.cronService)
		al.toolsRegistry.Register(cronTool)
	}
}

func (al *AgentLoop) Run(ctx context.Context) error {
	al.mu.Lock()
	al.running = true
	al.mu.Unlock()

	logrus.Info("Agent loop started")

	msgCh := make(chan *bus.InboundMessage, 10)

	go func() {
		for {
			msg, err := al.bus.ConsumeInbound(ctx)
			if err != nil {
				close(msgCh)
				return
			}
			if msg != nil {
				msgCh <- msg
			}
		}
	}()

	for {
		al.mu.RLock()
		running := al.running
		al.mu.RUnlock()

		if !running {
			logrus.Info("Agent loop exiting...")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgCh:
			if !ok {
				logrus.Info("Message channel closed, exiting...")
				return nil
			}
			if msg != nil {
				go al.processMessage(ctx, msg)
			}
		}
	}
}

func (al *AgentLoop) Stop() {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.running = false
	logrus.Info("Agent loop stopping")
}

func (al *AgentLoop) processMessage(ctx context.Context, msg *bus.InboundMessage) {
	preview := msg.Content
	if len(preview) > 80 {
		preview = preview[:80] + "..."
	}

	logrus.Infof("Processing message from %s:%s: %s", msg.Channel, msg.SenderID, preview)

	sess := al.sessionManager.GetOrCreate(msg.SessionKey())

	response, err := al.processDirect(ctx, msg.Content, msg.SessionKey(), msg.Channel, msg.ChatID, msg.Media)
	if err != nil {
		logrus.Errorf("Error processing message: %v", err)
		response = fmt.Sprintf("Sorry, I encountered an error: %v", err)
	}

	sess.AddMessage("user", msg.Content, nil)
	sess.AddMessage("assistant", response, nil)
	al.sessionManager.Save(sess)

	outbound := &bus.OutboundMessage{
		Channel:  msg.Channel,
		ChatID:   msg.ChatID,
		Content:  response,
		Metadata: msg.Metadata,
	}

	al.bus.PublishOutbound(outbound)
}

func (al *AgentLoop) ProcessDirect(ctx context.Context, content, sessionKey, channel, chatID string, media []string) (string, error) {
	return al.processDirect(ctx, content, sessionKey, channel, chatID, media)
}

func (al *AgentLoop) processDirect(ctx context.Context, content, sessionKey, channel, chatID string, media []string) (string, error) {
	sess := al.sessionManager.GetOrCreate(sessionKey)

	messages := al.context.BuildMessages(
		sess.GetHistory(50),
		content,
		nil,
		media,
		channel,
		chatID,
	)

	iteration := 0
	var finalContent string

	for iteration < al.maxIterations {
		iteration++

		response, err := al.provider.Chat(ctx, messages, ConvertToToolDefinitions(al.toolsRegistry.GetDefinitions()), al.model, 4096, 0.7)
		if err != nil {
			return "", fmt.Errorf("LLM error: %w", err)
		}

		if response.HasToolCalls() {
			toolCalls := make([]openai.ToolCall, 0, len(response.ToolCalls))
			for _, tc := range response.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: string(argsJSON),
					},
				})
			}

			messages = al.context.AddAssistantMessage(messages, response.Content, toolCalls, response.ReasoningContent)

			for _, toolCall := range response.ToolCalls {
				argsJSON, _ := json.Marshal(toolCall.Arguments)
				logrus.Infof("Tool call: %s(%s)", toolCall.Name, string(argsJSON)[:min(200, len(argsJSON))])

				result, err := al.toolsRegistry.Execute(ctx, toolCall.Name, toolCall.Arguments)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				messages = al.context.AddToolResult(messages, toolCall.ID, toolCall.Name, result)
			}
		} else {
			finalContent = response.Content
			break
		}
	}

	if finalContent == "" {
		finalContent = "I've completed processing but have no response to give."
	}

	preview := finalContent
	if len(preview) > 120 {
		preview = preview[:120] + "..."
	}
	logrus.Infof("Response to %s:%s: %s", channel, chatID, preview)

	return finalContent, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
