package tools

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/kbinani/screenshot"

	"github.com/nanobotgo/bus"
	"github.com/sirupsen/logrus"
)

type ScreenshotTool struct {
	sendCallback   func(*bus.OutboundMessage) error
	defaultChannel string
	defaultChatID  string
	tempDir        string
}

func NewScreenshotTool(sendCallback func(*bus.OutboundMessage) error, defaultChannel, defaultChatID, tempDir string) *ScreenshotTool {
	return &ScreenshotTool{
		sendCallback:   sendCallback,
		defaultChannel: defaultChannel,
		defaultChatID:  defaultChatID,
		tempDir:        tempDir,
	}
}

func (t *ScreenshotTool) SetContext(channel, chatID string) {
	t.defaultChannel = channel
	t.defaultChatID = chatID
}

func (t *ScreenshotTool) Name() string {
	return "screenshot"
}

func (t *ScreenshotTool) Description() string {
	return "Capture a screenshot of the computer screen. Use this when the user wants to take a screenshot or screen capture."
}

func (t *ScreenshotTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *ScreenshotTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	img, err := captureScreen()
	if err != nil {
		return "", fmt.Errorf("failed to capture screen: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode screenshot: %w", err)
	}

	filename := filepath.Join(t.tempDir, fmt.Sprintf("screenshot_%d.png", time.Now().UnixMilli()))
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	logrus.Infof("Screenshot saved to %s", filename)

	channel := t.defaultChannel
	chatID := t.defaultChatID

	if channel == "" || chatID == "" {
		return "", fmt.Errorf("no target channel/chat specified")
	}

	if t.sendCallback == nil {
		return "", fmt.Errorf("message sending not configured")
	}

	msg := &bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: "",
		Media:   []string{filename},
	}

	if err := t.sendCallback(msg); err != nil {
		return "", fmt.Errorf("error sending screenshot: %w", err)
	}

	return "Screenshot captured and sent", nil
}

func captureScreen() (image.Image, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}
	return img, nil
}
