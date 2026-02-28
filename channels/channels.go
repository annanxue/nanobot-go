package channels

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/config"
	"github.com/nanobotgo/utils"
)

type BaseChannelImpl struct {
	name    string
	config  interface{}
	bus     *bus.MessageBus
	running bool
	mu      sync.RWMutex
}

func NewBaseChannel(name string, cfg interface{}, bus *bus.MessageBus) *BaseChannelImpl {
	return &BaseChannelImpl{
		name:    name,
		config:  cfg,
		bus:     bus,
		running: false,
	}
}

func (bc *BaseChannelImpl) Name() string {
	return bc.name
}

func (bc *BaseChannelImpl) Start(ctx context.Context) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.running = true
	utils.Log.Infof("%s channel started", bc.name)
	return nil
}

func (bc *BaseChannelImpl) Stop(ctx context.Context) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.running = false
	utils.Log.Infof("%s channel stopped", bc.name)
	return nil
}

func (bc *BaseChannelImpl) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	utils.Log.Infof("Sending message to %s:%s", msg.Channel, msg.ChatID)
	return nil
}

func (bc *BaseChannelImpl) IsRunning() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.running
}

func (bc *BaseChannelImpl) IsAllowed(senderID string, allowFrom []string) bool {
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

func (bc *BaseChannelImpl) HandleMessage(senderID, chatID, content string, media []string, metadata map[string]interface{}, allowFrom []string) error {
	if !bc.IsAllowed(senderID, allowFrom) {
		utils.Log.Warnf("Access denied for sender %s on channel %s", senderID, bc.name)
		return nil
	}

	msg := &bus.InboundMessage{
		Channel:  bc.name,
		SenderID: senderID,
		ChatID:   chatID,
		Content:  content,
		Media:    media,
		Metadata: metadata,
	}

	bc.bus.PublishInbound(msg)
	return nil
}

type TelegramChannel struct {
	*BaseChannelImpl
	config *config.TelegramConfig
}

func NewTelegramChannel(cfg *config.TelegramConfig, bus *bus.MessageBus) (*TelegramChannel, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("telegram token not configured")
	}

	base := NewBaseChannel("telegram", cfg, bus)
	return &TelegramChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type WhatsAppChannel struct {
	*BaseChannelImpl
	config *config.WhatsAppConfig
}

func NewWhatsAppChannel(cfg *config.WhatsAppConfig, bus *bus.MessageBus) (*WhatsAppChannel, error) {
	base := NewBaseChannel("whatsapp", cfg, bus)
	return &WhatsAppChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type DiscordChannel struct {
	*BaseChannelImpl
	config *config.DiscordConfig
}

func NewDiscordChannel(cfg *config.DiscordConfig, bus *bus.MessageBus) (*DiscordChannel, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("discord token not configured")
	}

	base := NewBaseChannel("discord", cfg, bus)
	return &DiscordChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type MochatChannel struct {
	*BaseChannelImpl
	config *config.MochatConfig
}

func NewMochatChannel(cfg *config.MochatConfig, bus *bus.MessageBus) (*MochatChannel, error) {
	base := NewBaseChannel("mochat", cfg, bus)
	return &MochatChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type DingTalkChannel struct {
	*BaseChannelImpl
	config *config.DingTalkConfig
}

func NewDingTalkChannel(cfg *config.DingTalkConfig, bus *bus.MessageBus) (*DingTalkChannel, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("dingtalk credentials not configured")
	}

	base := NewBaseChannel("dingtalk", cfg, bus)
	return &DingTalkChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type EmailChannel struct {
	*BaseChannelImpl
	config *config.EmailConfig
}

func NewEmailChannel(cfg *config.EmailConfig, bus *bus.MessageBus) (*EmailChannel, error) {
	if !cfg.ConsentGranted {
		return nil, fmt.Errorf("email consent not granted")
	}

	base := NewBaseChannel("email", cfg, bus)
	return &EmailChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type SlackChannel struct {
	*BaseChannelImpl
	config *config.SlackConfig
}

func NewSlackChannel(cfg *config.SlackConfig, bus *bus.MessageBus) (*SlackChannel, error) {
	base := NewBaseChannel("slack", cfg, bus)
	return &SlackChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}

type QQChannel struct {
	*BaseChannelImpl
	config *config.QQConfig
}

func NewQQChannel(cfg *config.QQConfig, bus *bus.MessageBus) (*QQChannel, error) {
	if cfg.AppID == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("qq credentials not configured")
	}

	base := NewBaseChannel("qq", cfg, bus)
	return &QQChannel{
		BaseChannelImpl: base,
		config:          cfg,
	}, nil
}
