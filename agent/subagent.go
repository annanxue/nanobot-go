package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/providers"
	"github.com/nanobotgo/tools"
	"github.com/sirupsen/logrus"
)

type SubagentManager struct {
	provider          providers.LLMProvider
	workspace         string
	bus               *bus.MessageBus
	model             string
	braveAPIKey       string
	execConfig        *ExecToolConfig
	restrictWorkspace bool
	runningTasks      map[string]context.CancelFunc
}

func NewSubagentManager(
	provider providers.LLMProvider,
	workspace string,
	bus *bus.MessageBus,
	model string,
	braveAPIKey string,
	execConfig *ExecToolConfig,
	restrictWorkspace bool,
) *SubagentManager {
	if execConfig == nil {
		execConfig = &ExecToolConfig{Timeout: 60}
	}

	return &SubagentManager{
		provider:          provider,
		workspace:         workspace,
		bus:               bus,
		model:             model,
		braveAPIKey:       braveAPIKey,
		execConfig:        execConfig,
		restrictWorkspace: restrictWorkspace,
		runningTasks:      make(map[string]context.CancelFunc),
	}
}

func (sm *SubagentManager) Spawn(ctx context.Context, task, label, originChannel, originChatID string) (string, error) {
	taskID := time.Now().Format("20060102150405")[:8]
	displayLabel := label
	if displayLabel == "" {
		if len(task) > 30 {
			displayLabel = task[:30] + "..."
		} else {
			displayLabel = task
		}
	}

	taskCtx, cancel := context.WithCancel(context.Background())
	sm.runningTasks[taskID] = cancel

	go sm.runSubagent(taskCtx, taskID, task, displayLabel, originChannel, originChatID)

	logrus.Infof("Spawned subagent [%s]: %s", taskID, displayLabel)
	return fmt.Sprintf("Subagent [%s] started (id: %s). I'll notify you when it completes.", displayLabel, taskID), nil
}

func (sm *SubagentManager) runSubagent(ctx context.Context, taskID, task, label, originChannel, originChatID string) {
	logrus.Infof("Subagent [%s] starting task: %s", taskID, label)

	defer func() {
		delete(sm.runningTasks, taskID)
	}()

	toolsRegistry := tools.NewToolRegistry()

	allowedDir := ""
	if sm.restrictWorkspace {
		allowedDir = sm.workspace
	}

	toolsRegistry.Register(tools.NewReadFileTool(allowedDir))
	toolsRegistry.Register(tools.NewWriteFileTool(allowedDir))
	toolsRegistry.Register(tools.NewListDirTool(allowedDir))
	toolsRegistry.Register(tools.NewExecTool(
		sm.execConfig.Timeout,
		sm.workspace,
		nil,
		nil,
		sm.restrictWorkspace,
	))
	toolsRegistry.Register(tools.NewWebSearchTool(sm.braveAPIKey, 5))
	toolsRegistry.Register(tools.NewWebFetchTool(50000))

	systemPrompt := sm.buildSubagentPrompt(task)
	messages := []map[string]interface{}{
		{"role": "system", "content": systemPrompt},
		{"role": "user", "content": task},
	}

	maxIterations := 15
	iteration := 0
	var finalResult string

	for iteration < maxIterations {
		iteration++

		response, err := sm.provider.Chat(ctx, ConvertToProviderMessages(messages), ConvertToToolDefinitions(toolsRegistry.GetDefinitions()), sm.model, 4096, 0.7)
		if err != nil {
			finalResult = fmt.Sprintf("Error: %v", err)
			break
		}

		if response.HasToolCalls() {
			toolCallDicts := make([]map[string]interface{}, 0, len(response.ToolCalls))
			for _, tc := range response.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				toolCallDicts = append(toolCallDicts, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": string(argsJSON),
					},
				})
			}

			messages = append(messages, map[string]interface{}{
				"role":       "assistant",
				"content":    response.Content,
				"tool_calls": toolCallDicts,
			})

			for _, toolCall := range response.ToolCalls {
				argsJSON, _ := json.Marshal(toolCall.Arguments)
				logrus.Debugf("Subagent [%s] executing: %s with arguments: %s", taskID, toolCall.Name, string(argsJSON))

				result, err := toolsRegistry.Execute(ctx, toolCall.Name, toolCall.Arguments)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				messages = append(messages, map[string]interface{}{
					"role":         "tool",
					"tool_call_id": toolCall.ID,
					"name":         toolCall.Name,
					"content":      result,
				})
			}
		} else {
			finalResult = response.Content
			break
		}
	}

	if finalResult == "" {
		finalResult = "Task completed but no final response was generated."
	}

	logrus.Infof("Subagent [%s] completed successfully", taskID)
	sm.announceResult(taskID, label, task, finalResult, originChannel, originChatID, "ok")
}

func (sm *SubagentManager) announceResult(taskID, label, task, result, originChannel, originChatID, status string) {
	statusText := "completed successfully"
	if status != "ok" {
		statusText = "failed"
	}

	announceContent := fmt.Sprintf(`[Subagent '%s' %s]

Task: %s

Result:
%s

Summarize this naturally for the user. Keep it brief (1-2 sentences). Do not mention technical details like "subagent" or task IDs.`, label, statusText, task, result)

	msg := &bus.InboundMessage{
		Channel:  "system",
		SenderID: "subagent",
		ChatID:   fmt.Sprintf("%s:%s", originChannel, originChatID),
		Content:  announceContent,
	}

	sm.bus.PublishInbound(msg)
	logrus.Debugf("Subagent [%s] announced result to %s:%s", taskID, originChannel, originChatID)
}

func (sm *SubagentManager) buildSubagentPrompt(task string) string {
	return fmt.Sprintf(`# Subagent

You are a subagent spawned by the main agent to complete a specific task.

## Your Task
%s

## Rules
1. Stay focused - complete only the assigned task, nothing else
2. Your final response will be reported back to the main agent
3. Do not initiate conversations or take on side tasks
4. Be concise but informative in your findings

## What You Can Do
- Read and write files in the workspace
- Execute shell commands
- Search the web and fetch web pages
- Complete the task thoroughly

## What You Cannot Do
- Send messages directly to users (no message tool available)
- Spawn other subagents
- Access the main agent's conversation history

## Workspace
Your workspace is at: %s

When you have completed the task, provide a clear summary of your findings or actions.`, task, sm.workspace)
}

func (sm *SubagentManager) GetRunningCount() int {
	return len(sm.runningTasks)
}
