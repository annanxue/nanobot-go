package agent

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

type ContextBuilder struct {
	workspace string
	memory    *MemoryStore
	skills    *SkillsLoader
}

func NewContextBuilder(workspace string) *ContextBuilder {
	return &ContextBuilder{
		workspace: workspace,
		memory:    NewMemoryStore(workspace),
		skills:    NewSkillsLoader(workspace),
	}
}

func (cb *ContextBuilder) BuildSystemPrompt(skillNames []string) string {
	var parts []string

	parts = append(parts, cb.getIdentity())

	bootstrap := cb.loadBootstrapFiles()
	if bootstrap != "" {
		parts = append(parts, bootstrap)
	}

	memory := cb.memory.GetMemoryContext()
	if memory != "" {
		parts = append(parts, "# Memory\n\n"+memory)
	}

	alwaysSkills := cb.skills.GetAlwaysSkills()
	if len(alwaysSkills) > 0 {
		alwaysContent := cb.skills.LoadSkillsForContext(alwaysSkills)
		if alwaysContent != "" {
			parts = append(parts, "# Active Skills\n\n"+alwaysContent)
		}
	}

	skillsSummary := cb.skills.BuildSkillsSummary()
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.
Skills with available="false" need dependencies installed first - you can try installing them with apt/brew.

%s`, skillsSummary))
	}

	return strings.Join(parts, "\n\n---\n\n")
}

func (cb *ContextBuilder) getIdentity() string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	workspacePath := filepath.Clean(cb.workspace)

	system := runtime.GOOS
	runtimeStr := fmt.Sprintf("%s %s, Go %s", system, runtime.GOARCH, runtime.Version())
	if system == "darwin" {
		runtimeStr = fmt.Sprintf("macOS %s, Go %s", runtime.GOARCH, runtime.Version())
	}

	return fmt.Sprintf(`# nanobot 🐈

You are nanobot, a helpful AI assistant. You have access to tools that allow you to:
- Read, write, and edit files
- Execute shell commands
- Search the web and fetch web pages
- Send messages to users on chat channels
- Spawn subagents for complex background tasks

## Current Time
%s

## Runtime
%s

## Workspace
Your workspace is at: %s
- Memory files: %s/memory/MEMORY.md
- Daily notes: %s/memory/2006-01-02.md
- Custom skills: %s/skills/{skill-name}/SKILL.md

IMPORTANT: When responding to direct questions or conversations, reply directly with your text response.
Only use the 'message' tool when you need to send a message to a specific chat channel (like WhatsApp).
For normal conversation, just respond with text - do not call the message tool.

Always be helpful, accurate, and concise. When using tools, explain what you're doing.
When remembering something, write to %s/memory/MEMORY.md`, now, runtimeStr, workspacePath, workspacePath, workspacePath, workspacePath, workspacePath)
}

func (cb *ContextBuilder) loadBootstrapFiles() string {
	bootstrapFiles := []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md", "IDENTITY.md"}
	var parts []string

	for _, filename := range bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)
		if _, err := os.Stat(filePath); err == nil {
			content, _ := os.ReadFile(filePath)
			parts = append(parts, fmt.Sprintf("## %s\n\n%s", filename, string(content)))
		}
	}

	return strings.Join(parts, "\n\n")
}

func (cb *ContextBuilder) BuildMessages(history []map[string]interface{}, currentMessage string, skillNames []string, media []string, channel, chatID string) []openai.ChatCompletionMessage {
	messages := []openai.ChatCompletionMessage{}

	systemPrompt := cb.BuildSystemPrompt(skillNames)
	if channel != "" && chatID != "" {
		systemPrompt += fmt.Sprintf("\n\n## Current Session\nChannel: %s\nChat ID: %s", channel, chatID)
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	messages = append(messages, ConvertToProviderMessages(history)...)

	userContent := cb.buildUserContent(currentMessage, media)
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    "user",
		Content: userContent.(string),
	})

	return messages
}

func (cb *ContextBuilder) buildUserContent(text string, media []string) interface{} {
	if len(media) == 0 {
		return text
	}

	var images []openai.ChatMessagePart
	for _, path := range media {
		filepath.Clean(path) // Clean path but don't use the result
		mimeType := detectMimeType(path)
		if _, err := os.Stat(path); err != nil || mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
			continue
		}

		content, _ := os.ReadFile(path)
		b64 := base64.StdEncoding.EncodeToString(content)
		images = append(images, openai.ChatMessagePart{
			Type: "image_url",
			ImageURL: &openai.ChatMessageImageURL{
				URL: fmt.Sprintf("data:%s;base64,%s", mimeType, b64),
			},
		})
	}

	if len(images) == 0 {
		return text
	}

	result := make([]interface{}, 0, len(images)+1)
	for _, img := range images {
		result = append(result, img)
	}
	result = append(result, openai.ChatMessagePart{
		Type: "text",
		Text: text,
	})
	return result
}

func (cb *ContextBuilder) AddToolResult(messages []openai.ChatCompletionMessage, toolCallID, toolName, result string) []openai.ChatCompletionMessage {
	messages = append(messages, openai.ChatCompletionMessage{
		Role:       "tool",
		ToolCallID: toolCallID,
		Name:       toolName,
		Content:    result,
	})
	return messages
}

func (cb *ContextBuilder) AddAssistantMessage(messages []openai.ChatCompletionMessage, content string, toolCalls []openai.ToolCall, reasoningContent string) []openai.ChatCompletionMessage {
	msg := openai.ChatCompletionMessage{
		Role:    "assistant",
		Content: content,
	}

	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	if reasoningContent != "" {
		msg.ReasoningContent = reasoningContent
	}

	messages = append(messages, msg)
	return messages
}

func detectMimeType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
	}
	return mimeTypes[ext]
}
