package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/sirupsen/logrus"

	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/config"
)

// MSG_TYPE_MAP maps message types to display strings
var MSG_TYPE_MAP = map[string]string{
	"image":   "[image]",
	"audio":   "[audio]",
	"file":    "[file]",
	"sticker": "[sticker]",
}

// FeishuChannel implements the BaseChannel interface for Feishu/Lark
// Uses WebSocket long connection to receive events - no public IP required

type FeishuChannel struct {
	*BaseChannelImpl
	client          *lark.Client
	wsClient        *larkws.Client
	processedMsgIDs map[string]struct{}
	msgIDMutex      sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewFeishuChannel creates a new FeishuChannel instance
func NewFeishuChannel(cfg *config.FeishuConfig, bus *bus.MessageBus) (*FeishuChannel, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("feishu app_id and app_secret not configured")
	}

	base := NewBaseChannel("feishu", cfg, bus)
	return &FeishuChannel{
		BaseChannelImpl: base,
		processedMsgIDs: make(map[string]struct{}),
	}, nil
}

// Start starts the Feishu bot with WebSocket long connection
func (fc *FeishuChannel) Start(ctx context.Context) error {
	// Call base Start method
	if err := fc.BaseChannelImpl.Start(ctx); err != nil {
		return err
	}

	// Get Feishu config
	feishuCfg := fc.config.(*config.FeishuConfig)

	// Create event dispatcher
	eventDispatcher := dispatcher.NewEventDispatcher("", "")

	// Register message receive handler
	eventDispatcher.OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
		fc.handleMessage(event)
		return nil
	})

	// Create client for receiving events
	fc.client = lark.NewClient(
		feishuCfg.AppID,
		feishuCfg.AppSecret,
		lark.WithLogLevel(larkcore.LogLevelInfo),
	)
	/**
	* 启动长连接，并注册事件处理器。
	* Start long connection and register event handler.
	 */
	fc.wsClient = larkws.NewClient(feishuCfg.AppID, feishuCfg.AppSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	go func() {
		err := fc.wsClient.Start(ctx) // 会阻塞，直到上下文取消
		if err != nil {
			logrus.Errorf("Failed to start WebSocket client: %v", err)
		}
	}()

	// Create context for cancellation
	fc.ctx, fc.cancel = context.WithCancel(ctx)

	logrus.Info("Feishu bot started with WebSocket long connection")
	logrus.Info("No public IP required - using WebSocket to receive events")

	return nil
}

// Stop stops the Feishu bot
func (fc *FeishuChannel) Stop(ctx context.Context) error {
	// Call base Stop method
	if err := fc.BaseChannelImpl.Stop(ctx); err != nil {
		return err
	}

	// Cancel context
	if fc.cancel != nil {
		fc.cancel()
	}

	// Stop WebSocket client
	// WebSocket client will be stopped when context is canceled

	logrus.Info("Feishu bot stopped")
	return nil
}

// Send sends a message through Feishu
func (fc *FeishuChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	if !fc.running || fc.client == nil {
		logrus.Warn("Feishu API client not initialized")
		return fmt.Errorf("feishu API client not initialized")
	}

	// Determine receive_id_type based on chat_id format
	// open_id starts with "ou_", chat_id starts with "oc_"
	var receiveIDType string
	if strings.HasPrefix(msg.ChatID, "oc_") {
		receiveIDType = "chat_id"
	} else {
		receiveIDType = "open_id"
	}

	// Build card with markdown + table support
	elements := fc.buildCardElements(msg.Content)
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"elements": elements,
	}

	// Serialize card to JSON
	cardContent, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	// Build request body
	reqBody := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(msg.ChatID).
		MsgType("interactive").
		Content(string(cardContent)).
		Build()

	// Build request
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(reqBody).
		Build()

	// Send message
	resp, err := fc.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send Feishu message: %w", err)
	}

	if !resp.Success() {
		logID := resp.RequestId()
		return fmt.Errorf("failed to send Feishu message: code=%d, msg=%s, log_id=%s",
			resp.Code, resp.Msg, logID)
	}

	logrus.Debugf("Feishu message sent to %s", msg.ChatID)
	return nil
}

// addReaction adds a reaction to a Feishu message
func (fc *FeishuChannel) addReaction(ctx context.Context, messageID, emojiType string) error {
	if !fc.running || fc.client == nil {
		logrus.Warn("Feishu API client not initialized")
		return fmt.Errorf("feishu API client not initialized")
	}

	if messageID == "" {
		return fmt.Errorf("message ID is required")
	}

	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(larkim.NewEmojiBuilder().
				EmojiType(emojiType).
				Build()).
			Build()).
		Build()

	// 发起请求
	resp, err := fc.client.Im.V1.MessageReaction.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	if !resp.Success() {
		return fmt.Errorf("failed to add reaction: %s", resp.Msg)
	}
	logrus.Debugf("Added %s reaction to message %s", emojiType, messageID)
	return nil
}

// handleMessage handles incoming messages from Feishu
func (fc *FeishuChannel) handleMessage(event *larkim.P2MessageReceiveV1) {
	// Get message data
	msg := event.Event.Message
	messageID := ""
	if msg.MessageId != nil {
		messageID = *msg.MessageId
	}
	sender := event.Event.Sender

	// Deduplication check
	if messageID != "" && fc.isMessageProcessed(messageID) {
		return
	}
	if messageID != "" {
		fc.markMessageProcessed(messageID)
	}

	// Trim cache: keep most recent 500 when exceeds 1000
	fc.trimMessageCache()

	// Skip bot messages
	if sender != nil && sender.SenderType != nil && *sender.SenderType == "bot" {
		return
	}

	// Get sender and chat info
	senderID := ""
	if sender != nil && sender.SenderId != nil && sender.SenderId.OpenId != nil {
		senderID = *sender.SenderId.OpenId
	}
	if senderID == "" {
		senderID = "unknown"
	}
	chatID := ""
	if msg.ChatId != nil {
		chatID = *msg.ChatId
	}
	chatType := ""
	if msg.ChatType != nil {
		chatType = *msg.ChatType
	}
	msgType := ""
	if msg.MessageType != nil {
		msgType = *msg.MessageType
	}

	// Add reaction to indicate "seen"
	if messageID != "" {
		err := fc.addReaction(fc.ctx, messageID, "THUMBSUP")
		if err != nil {
			logrus.Errorf("Failed to add reaction: %v", err)
		}
	}

	// Parse message content
	var content string
	if msgType == "text" {
		if msg.Content != nil {
			var textContent map[string]string
			if err := json.Unmarshal([]byte(*msg.Content), &textContent); err == nil {
				content = textContent["text"]
			} else {
				content = *msg.Content
			}
		}
	} else {
		if display, ok := MSG_TYPE_MAP[msgType]; ok {
			content = display
		} else {
			content = fmt.Sprintf("[%s]", msgType)
		}
	}

	if content == "" {
		return
	}

	// Forward to message bus
	replyTo := chatID
	if chatType == "p2p" {
		replyTo = senderID
	}

	metadata := map[string]interface{}{
		"message_id": messageID,
		"chat_type":  chatType,
		"msg_type":   msgType,
	}

	// Get Feishu config
	feishuCfg := fc.config.(*config.FeishuConfig)

	// Check if sender is allowed
	if !fc.IsAllowed(senderID, feishuCfg.AllowFrom) {
		logrus.Warnf("Access denied for sender %s on feishu channel", senderID)
		return
	}

	// Create inbound message
	inboundMsg := &bus.InboundMessage{
		Channel:  "feishu",
		SenderID: senderID,
		ChatID:   replyTo,
		Content:  content,
		Media:    []string{},
		Metadata: metadata,
	}

	// Publish to bus
	fc.bus.PublishInbound(inboundMsg)
}

// isMessageProcessed checks if a message has been processed
func (fc *FeishuChannel) isMessageProcessed(messageID string) bool {
	fc.msgIDMutex.Lock()
	defer fc.msgIDMutex.Unlock()
	_, ok := fc.processedMsgIDs[messageID]
	return ok
}

// markMessageProcessed marks a message as processed
func (fc *FeishuChannel) markMessageProcessed(messageID string) {
	fc.msgIDMutex.Lock()
	defer fc.msgIDMutex.Unlock()
	fc.processedMsgIDs[messageID] = struct{}{}
}

// trimMessageCache trims the message ID cache to keep most recent 500
func (fc *FeishuChannel) trimMessageCache() {
	fc.msgIDMutex.Lock()
	defer fc.msgIDMutex.Unlock()

	if len(fc.processedMsgIDs) <= 1000 {
		return
	}

	// Create a new map with capacity 500
	newCache := make(map[string]struct{}, 500)
	count := 0

	// Add most recent entries (in practice, this will be random due to map iteration)
	// For a more accurate LRU, we would use a different data structure
	for msgID := range fc.processedMsgIDs {
		if count < 500 {
			newCache[msgID] = struct{}{}
			count++
		} else {
			break
		}
	}

	fc.processedMsgIDs = newCache
}

// tableRegex matches markdown tables
var tableRegex = regexp.MustCompile(`((?:^[ \t]*\|.+\|[ \t]*\n)(?:^[ \t]*\|[\-:\s|]+\|[ \t]*\n)(?:^[ \t]*\|.+\|[ \t]*\n?)+)`)

// buildCardElements splits content into markdown + table elements for Feishu card
func (fc *FeishuChannel) buildCardElements(content string) []map[string]interface{} {
	elements := []map[string]interface{}{}
	lastEnd := 0

	// Find all tables
	matches := tableRegex.FindAllStringIndex(content, -1)
	for _, match := range matches {
		start, end := match[0], match[1]

		// Add content before table
		before := content[lastEnd:start]
		if before != "" {
			elements = append(elements, map[string]interface{}{
				"tag":     "markdown",
				"content": before,
			})
		}

		// Add table
		tableContent := content[start:end]
		if table := fc.parseMDTable(tableContent); table != nil {
			elements = append(elements, table)
		} else {
			// If parsing fails, add as markdown
			elements = append(elements, map[string]interface{}{
				"tag":     "markdown",
				"content": tableContent,
			})
		}

		lastEnd = end
	}

	// Add remaining content
	remaining := content[lastEnd:]
	if remaining != "" {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": remaining,
		})
	}

	// If no elements, add empty markdown
	if len(elements) == 0 {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": content,
		})
	}

	return elements
}

// parseMDTable parses a markdown table into a Feishu table element
func (fc *FeishuChannel) parseMDTable(tableText string) map[string]interface{} {
	// Split table into lines
	lines := []string{}
	for _, line := range strings.Split(tableText, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	// Need at least header, separator, and one data row
	if len(lines) < 3 {
		return nil
	}

	// Parse headers
	headers := []string{}
	headerLine := strings.TrimSpace(lines[0])
	headerLine = strings.Trim(headerLine, "|")
	for _, header := range strings.Split(headerLine, "|") {
		headers = append(headers, strings.TrimSpace(header))
	}

	// Parse data rows
	rows := []map[string]string{}
	for i := 2; i < len(lines); i++ {
		rowLine := strings.TrimSpace(lines[i])
		rowLine = strings.Trim(rowLine, "|")
		rowData := map[string]string{}
		cells := strings.Split(rowLine, "|")
		for j, cell := range cells {
			if j < len(headers) {
				colKey := fmt.Sprintf("c%d", j)
				rowData[colKey] = strings.TrimSpace(cell)
			}
		}
		rows = append(rows, rowData)
	}

	// Create columns
	columns := []map[string]interface{}{}
	for i, header := range headers {
		columns = append(columns, map[string]interface{}{
			"tag":          "column",
			"name":         fmt.Sprintf("c%d", i),
			"display_name": header,
			"width":        "auto",
		})
	}

	// Create table
	table := map[string]interface{}{
		"tag":       "table",
		"page_size": len(rows) + 1,
		"columns":   columns,
		"rows":      rows,
	}

	return table
}
