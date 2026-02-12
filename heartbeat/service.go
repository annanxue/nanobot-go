package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DefaultHeartbeatIntervalS = 30 * 60
	HeartbeatPrompt           = `Read HEARTBEAT.md in your workspace (if it exists).
Follow any instructions or tasks listed there.
If nothing needs attention, reply with just: HEARTBEAT_OK`
	HeartbeatOKToken = "HEARTBEAT_OK"
)

type HeartbeatService struct {
	workspace   string
	onHeartbeat func(string) (string, error)
	intervalS   int
	enabled     bool
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewHeartbeatService(workspace string, onHeartbeat func(string) (string, error), intervalS int, enabled bool) *HeartbeatService {
	return &HeartbeatService{
		workspace:   workspace,
		onHeartbeat: onHeartbeat,
		intervalS:   intervalS,
		enabled:     enabled,
		running:     false,
	}
}

func (hs *HeartbeatService) HeartbeatFile() string {
	return filepath.Join(hs.workspace, "HEARTBEAT.md")
}

func (hs *HeartbeatService) readHeartbeatFile() string {
	path := hs.HeartbeatFile()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(content)
}

func isHeartbeatEmpty(content string) bool {
	if content == "" {
		return true
	}

	skipPatterns := []string{"- [ ]", "* [ ]", "- [x]", "* [x]"}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "<!--") {
			continue
		}
		for _, pattern := range skipPatterns {
			if line == pattern {
				continue
			}
		}
		return false
	}

	return true
}

func (hs *HeartbeatService) Start() error {
	if !hs.enabled {
		logrus.Info("Heartbeat disabled")
		return nil
	}

	hs.ctx, hs.cancel = context.WithCancel(context.Background())
	hs.running = true

	go hs.runLoop()

	logrus.Infof("Heartbeat started (every %ds)", hs.intervalS)
	return nil
}

func (hs *HeartbeatService) Stop() {
	if !hs.running {
		return
	}

	hs.running = false
	if hs.cancel != nil {
		hs.cancel()
	}
}

func (hs *HeartbeatService) runLoop() {
	ticker := time.NewTicker(time.Duration(hs.intervalS) * time.Second)
	defer ticker.Stop()

	for hs.running {
		select {
		case <-ticker.C:
			hs.tick()
		case <-hs.ctx.Done():
			return
		}
	}
}

func (hs *HeartbeatService) tick() {
	content := hs.readHeartbeatFile()

	if isHeartbeatEmpty(content) {
		logrus.Debug("Heartbeat: no tasks (HEARTBEAT.md empty)")
		return
	}

	logrus.Info("Heartbeat: checking for tasks...")

	if hs.onHeartbeat != nil {
		response, err := hs.onHeartbeat(HeartbeatPrompt)
		if err != nil {
			logrus.Errorf("Heartbeat execution failed: %v", err)
			return
		}

		responseUpper := strings.ToUpper(response)
		responseUpper = strings.ReplaceAll(responseUpper, "_", "")
		tokenUpper := strings.ToUpper(HeartbeatOKToken)
		tokenUpper = strings.ReplaceAll(tokenUpper, "_", "")

		if strings.Contains(responseUpper, tokenUpper) {
			logrus.Info("Heartbeat: OK (no action needed)")
		} else {
			logrus.Info("Heartbeat: completed task")
		}
	}
}

func (hs *HeartbeatService) TriggerNow() (string, error) {
	if hs.onHeartbeat != nil {
		return hs.onHeartbeat(HeartbeatPrompt)
	}
	return "", nil
}
