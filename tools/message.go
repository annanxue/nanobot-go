package tools

import (
	"context"
	"fmt"

	"github.com/nanobotgo/bus"
)

type MessageTool struct {
	sendCallback    func(*bus.OutboundMessage) error
	defaultChannel string
	defaultChatID  string
}

func NewMessageTool(sendCallback func(*bus.OutboundMessage) error, defaultChannel, defaultChatID string) *MessageTool {
	return &MessageTool{
		sendCallback:    sendCallback,
		defaultChannel: defaultChannel,
		defaultChatID:  defaultChatID,
	}
}

func (t *MessageTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *MessageTool) SetSendCallback(callback func(*bus.OutboundMessage) error) {
	t.sendCallback = callback
}

func (t *MessageTool) Name() string {
	return "message"
}

func (t *MessageTool) Description() string {
	return "Send a message to the user. Use this when you want to communicate something."
}

func (t *MessageTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The message content to send",
			},
			"channel": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target channel (telegram, discord, etc.)",
			},
			"chat_id": map[string]interface{}{
				"type":        "string",
				"description": "Optional: target chat/user ID",
			},
		},
		"required": []string{"content"},
	}
}

func (t *MessageTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}

	channel := t.defaultChannel
	if ch, ok := params["channel"].(string); ok {
		channel = ch
	}

	chatID := t.defaultChatID
	if cid, ok := params["chat_id"].(string); ok {
		chatID = cid
	}

	if channel == "" || chatID == "" {
		return "", fmt.Errorf("no target channel/chat specified")
	}

	if t.sendCallback == nil {
		return "", fmt.Errorf("message sending not configured")
	}

	msg := &bus.OutboundMessage{
		Channel:  channel,
		ChatID:   chatID,
		Content:  content,
	}

	if err := t.sendCallback(msg); err != nil {
		return "", fmt.Errorf("error sending message: %w", err)
	}

	return fmt.Sprintf("Message sent to %s:%s", channel, chatID), nil
}
