package agent

import (
	"context"
	"fmt"

	"github.com/nanobotgo/cron"
)

type CronTool struct {
	cronService *cron.CronService
	channel     string
	chatID      string
}

func NewCronTool(cronService *cron.CronService) *CronTool {
	return &CronTool{
		cronService: cronService,
		channel:     "",
		chatID:      "",
	}
}

func (ct *CronTool) SetContext(channel, chatID string) {
	ct.channel = channel
	ct.chatID = chatID
}

func (ct *CronTool) Name() string {
	return "cron"
}

func (ct *CronTool) Description() string {
	return "Schedule reminders and recurring tasks. Actions: add, list, remove."
}

func (ct *CronTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"add", "list", "remove"},
				"description": "Action to perform",
			},
			"message": map[string]interface{}{
				"type":        "string",
				"description": "Reminder message (for add)",
			},
			"every_seconds": map[string]interface{}{
				"type":        "number",
				"description": "Interval in seconds (for recurring tasks)",
			},
			"cron_expr": map[string]interface{}{
				"type":        "string",
				"description": "Cron expression like '0 9 * * *' (for scheduled tasks)",
			},
			"job_id": map[string]interface{}{
				"type":        "string",
				"description": "Job ID (for remove)",
			},
		},
		"required": []string{"action"},
	}
}

func (ct *CronTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	action, ok := params["action"].(string)
	if !ok {
		return "", fmt.Errorf("action is required")
	}

	switch action {
	case "add":
		return ct.addJob(params)
	case "list":
		return ct.listJobs()
	case "remove":
		return ct.removeJob(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (ct *CronTool) addJob(params map[string]interface{}) (string, error) {
	message, ok := params["message"].(string)
	if !ok || message == "" {
		return "", fmt.Errorf("message is required for add")
	}

	if ct.channel == "" || ct.chatID == "" {
		return "", fmt.Errorf("no session context (channel/chat_id)")
	}

	var schedule cron.CronSchedule

	if everySeconds, ok := params["every_seconds"].(float64); ok {
		everyMs := int64(everySeconds * 1000)
		schedule = cron.CronSchedule{
			Kind:    cron.ScheduleKindEvery,
			EveryMs: &everyMs,
		}
	} else if cronExpr, ok := params["cron_expr"].(string); ok {
		schedule = cron.CronSchedule{
			Kind: cron.ScheduleKindCron,
			Expr: cronExpr,
		}
	} else {
		return "", fmt.Errorf("either every_seconds or cron_expr is required")
	}

	jobName := message
	if len(message) > 30 {
		jobName = message[:30]
	}

	job, err := ct.cronService.AddJob(
		jobName,
		schedule,
		message,
		true,
		ct.channel,
		ct.chatID,
		false,
	)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Created job '%s' (id: %s)", job.Name, job.ID), nil
}

func (ct *CronTool) listJobs() (string, error) {
	jobs := ct.cronService.ListJobs(false)
	if len(jobs) == 0 {
		return "No scheduled jobs.", nil
	}

	var lines []string
	lines = append(lines, "Scheduled jobs:")
	for _, job := range jobs {
		lines = append(lines, fmt.Sprintf("- %s (id: %s, %s)", job.Name, job.ID, job.Schedule.Kind))
	}

	return fmt.Sprintf("%s\n", lines[0]) + fmt.Sprintf("%s", lines[1]), nil
}

func (ct *CronTool) removeJob(params map[string]interface{}) (string, error) {
	jobID, ok := params["job_id"].(string)
	if !ok || jobID == "" {
		return "", fmt.Errorf("job_id is required for remove")
	}

	if ct.cronService.RemoveJob(jobID) {
		return fmt.Sprintf("Removed job %s", jobID), nil
	}

	return fmt.Sprintf("Job %s not found", jobID), nil
}
