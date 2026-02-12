package channels

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/config"
	"github.com/nanobotgo/session"
	"github.com/sirupsen/logrus"
)

type BaseChannel interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Send(ctx context.Context, msg *bus.OutboundMessage) error
	IsRunning() bool
}

type ChannelManager struct {
	config         *config.Config
	bus            *bus.MessageBus
	sessionManager *session.SessionManager
	channels       map[string]BaseChannel
	dispatchCtx    context.Context
	dispatchCancel context.CancelFunc
	running        bool
	mu             sync.RWMutex
}

func NewChannelManager(cfg *config.Config, bus *bus.MessageBus, sessionManager *session.SessionManager) *ChannelManager {
	return &ChannelManager{
		config:         cfg,
		bus:            bus,
		sessionManager: sessionManager,
		channels:       make(map[string]BaseChannel),
		running:        false,
	}
}

func (cm *ChannelManager) InitChannels() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.config.Channels.Telegram.Enabled {
		telegram, err := NewTelegramChannel(&cm.config.Channels.Telegram, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize Telegram channel: %v", err)
		} else {
			cm.channels["telegram"] = telegram
			logrus.Info("Telegram channel enabled")
		}
	}

	if cm.config.Channels.WhatsApp.Enabled {
		whatsapp, err := NewWhatsAppChannel(&cm.config.Channels.WhatsApp, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize WhatsApp channel: %v", err)
		} else {
			cm.channels["whatsapp"] = whatsapp
			logrus.Info("WhatsApp channel enabled")
		}
	}

	if cm.config.Channels.Discord.Enabled {
		discord, err := NewDiscordChannel(&cm.config.Channels.Discord, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize Discord channel: %v", err)
		} else {
			cm.channels["discord"] = discord
			logrus.Info("Discord channel enabled")
		}
	}

	if cm.config.Channels.Feishu.Enabled {
		feishu, err := NewFeishuChannel(&cm.config.Channels.Feishu, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize Feishu channel: %v", err)
		} else {
			cm.channels["feishu"] = feishu
			logrus.Info("Feishu channel enabled")
		}
	}

	if cm.config.Channels.Mochat.Enabled {
		mochat, err := NewMochatChannel(&cm.config.Channels.Mochat, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize Mochat channel: %v", err)
		} else {
			cm.channels["mochat"] = mochat
			logrus.Info("Mochat channel enabled")
		}
	}

	if cm.config.Channels.DingTalk.Enabled {
		dingtalk, err := NewDingTalkChannel(&cm.config.Channels.DingTalk, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize DingTalk channel: %v", err)
		} else {
			cm.channels["dingtalk"] = dingtalk
			logrus.Info("DingTalk channel enabled")
		}
	}

	if cm.config.Channels.Email.Enabled {
		email, err := NewEmailChannel(&cm.config.Channels.Email, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize Email channel: %v", err)
		} else {
			cm.channels["email"] = email
			logrus.Info("Email channel enabled")
		}
	}

	if cm.config.Channels.Slack.Enabled {
		slack, err := NewSlackChannel(&cm.config.Channels.Slack, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize Slack channel: %v", err)
		} else {
			cm.channels["slack"] = slack
			logrus.Info("Slack channel enabled")
		}
	}

	if cm.config.Channels.QQ.Enabled {
		qq, err := NewQQChannel(&cm.config.Channels.QQ, cm.bus)
		if err != nil {
			logrus.Warnf("Failed to initialize QQ channel: %v", err)
		} else {
			cm.channels["qq"] = qq
			logrus.Info("QQ channel enabled")
		}
	}

	return nil
}

func (cm *ChannelManager) StartAll(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.channels) == 0 {
		logrus.Warn("No channels enabled")
		return nil
	}

	cm.dispatchCtx, cm.dispatchCancel = context.WithCancel(ctx)
	cm.running = true

	go cm.dispatchOutbound()

	for name, channel := range cm.channels {
		logrus.Infof("Starting %s channel...", name)
		go func(name string, ch BaseChannel) {
			if err := ch.Start(ctx); err != nil {
				logrus.Errorf("Failed to start channel %s: %v", name, err)
			}
		}(name, channel)
	}

	return nil
}

func (cm *ChannelManager) StopAll(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	logrus.Info("Stopping all channels...")

	if cm.dispatchCancel != nil {
		cm.dispatchCancel()
	}

	for name, channel := range cm.channels {
		if err := channel.Stop(ctx); err != nil {
			logrus.Errorf("Error stopping %s: %v", name, err)
		} else {
			logrus.Infof("Stopped %s channel", name)
		}
	}

	cm.running = false
	return nil
}

func (cm *ChannelManager) dispatchOutbound() {
	logrus.Info("Outbound dispatcher started")

	for {
		select {
		case <-cm.dispatchCtx.Done():
			return
		default:
			msg, err := cm.bus.ConsumeOutbound(cm.dispatchCtx)
			if err != nil {
				continue
			}

			cm.mu.RLock()
			channel, exists := cm.channels[msg.Channel]
			cm.mu.RUnlock()

			if exists {
				if err := channel.Send(cm.dispatchCtx, msg); err != nil {
					logrus.Errorf("Error sending to %s: %v", msg.Channel, err)
				}
			} else {
				logrus.Warnf("Unknown channel: %s", msg.Channel)
			}
		}
	}
}

func (cm *ChannelManager) GetChannel(name string) BaseChannel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.channels[name]
}

func (cm *ChannelManager) GetStatus() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	status := make(map[string]interface{})
	for name, channel := range cm.channels {
		status[name] = map[string]interface{}{
			"enabled": true,
			"running": channel.IsRunning(),
		}
	}
	return status
}

func (cm *ChannelManager) EnabledChannels() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	names := make([]string, 0, len(cm.channels))
	for name := range cm.channels {
		names = append(names, name)
	}
	return names
}

func (cm *ChannelManager) IsAllowed(senderID string, allowFrom []string) bool {
	if len(allowFrom) == 0 {
		return true
	}

	senderStr := strings.TrimSpace(senderID)
	for _, allowed := range allowFrom {
		if senderStr == allowed {
			return true
		}
		if strings.Contains(senderStr, "|") {
			parts := strings.Split(senderStr, "|")
			for _, part := range parts {
				if strings.TrimSpace(part) == allowed {
					return true
				}
			}
		}
	}
	return false
}

func (cm *ChannelManager) HandleMessage(channel string, senderID, chatID, content string, media []string, metadata map[string]interface{}) error {
	cm.mu.RLock()
	_, exists := cm.channels[channel]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channel)
	}

	allowFrom := cm.getAllowList(channel)
	if !cm.IsAllowed(senderID, allowFrom) {
		logrus.Warnf("Access denied for sender %s on channel %s. Add them to allowFrom list in config.", senderID, channel)
		return nil
	}

	msg := &bus.InboundMessage{
		Channel:  channel,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  content,
		Media:    media,
		Metadata: metadata,
	}

	cm.bus.PublishInbound(msg)
	return nil
}

func (cm *ChannelManager) getAllowList(channel string) []string {
	switch channel {
	case "telegram":
		return cm.config.Channels.Telegram.AllowFrom
	case "whatsapp":
		return cm.config.Channels.WhatsApp.AllowFrom
	case "discord":
		return cm.config.Channels.Discord.AllowFrom
	case "feishu":
		return cm.config.Channels.Feishu.AllowFrom
	case "mochat":
		return cm.config.Channels.Mochat.AllowFrom
	case "dingtalk":
		return cm.config.Channels.DingTalk.AllowFrom
	case "email":
		return cm.config.Channels.Email.AllowFrom
	case "slack":
		return cm.config.Channels.Slack.DM.AllowFrom
	case "qq":
		return cm.config.Channels.QQ.AllowFrom
	default:
		return []string{}
	}
}
