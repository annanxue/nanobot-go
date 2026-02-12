package agent

import (
	"context"
	"fmt"
)

type SpawnTool struct {
	manager       *SubagentManager
	originChannel string
	originChatID  string
}

func NewSpawnTool(manager *SubagentManager) *SpawnTool {
	return &SpawnTool{
		manager:       manager,
		originChannel: "cli",
		originChatID:  "direct",
	}
}

func (st *SpawnTool) SetContext(channel, chatID string) {
	st.originChannel = channel
	st.originChatID = chatID
}

func (st *SpawnTool) Name() string {
	return "spawn"
}

func (st *SpawnTool) Description() string {
	return "Spawn a subagent to handle a task in the background. Use this for complex or time-consuming tasks that can run independently. The subagent will complete the task and report back when done."
}

func (st *SpawnTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task": map[string]interface{}{
				"type":        "string",
				"description": "The task for the subagent to complete",
			},
			"label": map[string]interface{}{
				"type":        "string",
				"description": "Optional short label for the task (for display)",
			},
		},
		"required": []string{"task"},
	}
}

func (st *SpawnTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	task, ok := params["task"].(string)
	if !ok {
		return "", fmt.Errorf("task is required")
	}

	label, _ := params["label"].(string)

	return st.manager.Spawn(ctx, task, label, st.originChannel, st.originChatID)
}
