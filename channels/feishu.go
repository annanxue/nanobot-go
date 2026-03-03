package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/config"
	"github.com/nanobotgo/utils"
)

// MSG_TYPE_MAP maps message types to display strings
var MSG_TYPE_MAP = map[string]string{
	"image":   "[image]",
	"audio":   "[audio]",
	"file":    "[file]",
	"sticker": "[sticker]",
}

// CustomLogger implements larkcore.Logger interface
type CustomLogger struct{}

// Debug logs debug messages
func (c *CustomLogger) Debug(ctx context.Context, args ...interface{}) {
	utils.Log.Debug(args...)
}

// Info logs info messages
func (c *CustomLogger) Info(ctx context.Context, args ...interface{}) {
	utils.Log.Info(args...)
}

// Warn logs warn messages
func (c *CustomLogger) Warn(ctx context.Context, args ...interface{}) {
	utils.Log.Warn(args...)
}

// Error logs error messages
func (c *CustomLogger) Error(ctx context.Context, args ...interface{}) {
	utils.Log.Error(args...)
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
		lark.WithLogger(&CustomLogger{}), // 使用自定义Logger
		lark.WithLogLevel(larkcore.LogLevelInfo),
	)
	/**
	* 启动长连接，并注册事件处理器。
	* Start long connection and register event handler.
	 */
	fc.wsClient = larkws.NewClient(feishuCfg.AppID, feishuCfg.AppSecret,
		larkws.WithEventHandler(eventDispatcher),
		larkws.WithLogger(&CustomLogger{}), // 使用自定义Logger
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	go func() {
		err := fc.wsClient.Start(ctx) // 会阻塞，直到上下文取消
		if err != nil {
			utils.Log.Errorf("Failed to start WebSocket client: %v", err)
		}
	}()

	// Create context for cancellation
	fc.ctx, fc.cancel = context.WithCancel(ctx)

	utils.Log.Info("Feishu bot started with WebSocket long connection")
	utils.Log.Info("No public IP required - using WebSocket to receive events")

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

	utils.Log.Info("Feishu bot stopped")
	return nil
}

// Send sends a message through Feishu
func (fc *FeishuChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	if !fc.running || fc.client == nil {
		utils.Log.Warn("Feishu API client not initialized")
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

	// Handle image message
	if len(msg.Media) > 0 {
		return fc.sendImageMessage(ctx, msg, receiveIDType)
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

	utils.Log.Debugf("Feishu message sent to %s", msg.ChatID)
	return nil
}

// sendImageMessage sends an image message through Feishu
func (fc *FeishuChannel) sendImageMessage(ctx context.Context, msg *bus.OutboundMessage, receiveIDType string) error {
	imagePath := msg.Media[0]

	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	imageReq := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(file).
			Build()).
		Build()

	imageResp, err := fc.client.Im.V1.Image.Create(ctx, imageReq)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	if !imageResp.Success() {
		return fmt.Errorf("failed to upload image: code=%d, msg=%s", imageResp.Code, imageResp.Msg)
	}

	imageKey := imageResp.Data.ImageKey

	messageImage := larkim.MessageImage{ImageKey: *imageKey}
	content, err := messageImage.String()
	if err != nil {
		return fmt.Errorf("failed to create image content: %w", err)
	}

	reqBody := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(msg.ChatID).
		MsgType("image").
		Content(content).
		Build()

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(reqBody).
		Build()

	resp, err := fc.client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send image message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("failed to send image message: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	utils.Log.Debugf("Feishu image sent to %s", msg.ChatID)
	return nil
}

// addReaction adds a reaction to a Feishu message
func (fc *FeishuChannel) addReaction(ctx context.Context, messageID, emojiType string) error {
	if !fc.running || fc.client == nil {
		utils.Log.Warn("Feishu API client not initialized")
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
	utils.Log.Debugf("Added %s reaction to message %s", emojiType, messageID)
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
			utils.Log.Errorf("Failed to add reaction: %v", err)
		}
	}

	// Parse message content
	var content string
	var media []string

	if msgType == "text" {
		if msg.Content != nil {
			var textContent map[string]string
			if err := json.Unmarshal([]byte(*msg.Content), &textContent); err == nil {
				content = textContent["text"]
			} else {
				content = *msg.Content
			}
		}
	} else if msgType == "image" && msg.Content != nil {
		imgContent, imgMedia, err := fc.handleImageMessage(*msg.Content, messageID)
		content = imgContent
		media = imgMedia
		if err != nil {
			utils.Log.Errorf("Failed to handle image message: %v", err)
			content = "[image]"
		}
	} else if msgType == "post" && msg.Content != nil {
		content, media = fc.handlePostMessage(*msg.Content, messageID)
	} else if (msgType == "audio" || msgType == "file" || msgType == "media") && msg.Content != nil {
		mediaContent, mediaPath, err := fc.handleMediaMessage(msgType, *msg.Content, messageID)
		content = mediaContent
		media = mediaPath
		if err != nil {
			utils.Log.Errorf("Failed to handle media message: %v", err)
		}
	} else if isShareCardType(msgType) {
		if msg.Content != nil {
			content = extractShareCardContent(*msg.Content, msgType)
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
		utils.Log.Warnf("Access denied for sender %s on feishu channel", senderID)
		return
	}

	// Create inbound message
	inboundMsg := &bus.InboundMessage{
		Channel:  "feishu",
		SenderID: senderID,
		ChatID:   replyTo,
		Content:  content,
		Media:    media,
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

func (fc *FeishuChannel) handleImageMessage(contentJSON string, messageID string) (string, []string, error) {
	var content map[string]string
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return "", nil, fmt.Errorf("failed to parse image content: %w", err)
	}

	imageKey := content["image_key"]
	if imageKey == "" || messageID == "" {
		return "", nil, nil
	}

	imagePath, err := fc.saveImage(messageID, imageKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download image: %w", err)
	}
	return imagePath, []string{imagePath}, nil
}

func (fc *FeishuChannel) handleMediaMessage(msgType string, contentJSON string, messageID string) (string, []string, error) {
	var content map[string]string
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return fmt.Sprintf("[%s: download failed]", msgType), nil, nil
	}

	var fileKey string
	switch msgType {
	case "audio":
		fileKey = content["file_key"]
	case "file":
		fileKey = content["file_key"]
	case "media":
		fileKey = content["file_key"]
	default:
		fileKey = content["file_key"]
	}

	if fileKey == "" || messageID == "" {
		return fmt.Sprintf("[%s]", msgType), nil, nil
	}

	filePath, err := fc.saveFile(messageID, fileKey, msgType)
	if err != nil {
		utils.Log.Errorf("Failed to download %s: %v", msgType, err)
		return fmt.Sprintf("[%s: download failed]", msgType), nil, nil
	}

	return fmt.Sprintf("[%s: %s]", msgType, filepath.Base(filePath)), []string{filePath}, nil
}

func (fc *FeishuChannel) saveImage(messageID string, imageKey string) (string, error) {
	imageData, err := fc.downloadImage(messageID, imageKey)
	if err != nil {
		return "", err
	}

	workspace := fc.getWorkspace()
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("feishu_image_%s.jpg", imageKey)
	imagePath := filepath.Join(workspace, filename)
	if err := os.WriteFile(imagePath, imageData, 0644); err != nil {
		return "", err
	}

	utils.Log.Infof("Saved feishu image to: %s", imagePath)
	return imagePath, nil
}

func (fc *FeishuChannel) saveFile(messageID string, fileKey string, fileType string) (string, error) {
	data, err := fc.downloadFile(messageID, fileKey, fileType)
	if err != nil {
		return "", err
	}

	workspace := fc.getWorkspace()
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return "", err
	}

	var ext string
	switch fileType {
	case "audio":
		ext = ".opus"
	case "media":
		ext = ".mp4"
	default:
		ext = ""
	}

	filename := fmt.Sprintf("feishu_%s_%s%s", fileType, fileKey, ext)
	filePath := filepath.Join(workspace, filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	utils.Log.Infof("Saved feishu %s to: %s", fileType, filePath)
	return filePath, nil
}

func (fc *FeishuChannel) downloadFile(messageID string, fileKey string, fileType string) ([]byte, error) {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(fileKey).
		Type(fileType).
		Build()

	resp, err := fc.client.Im.V1.MessageResource.Get(fc.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get file resource: %w", err)
	}

	if !resp.Success() {
		return nil, fmt.Errorf("failed to get file: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	if resp.File == nil {
		return nil, fmt.Errorf("file data is nil")
	}

	return io.ReadAll(resp.File)
}

func (fc *FeishuChannel) downloadImage(messageID string, imageKey string) ([]byte, error) {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(imageKey).
		Type("image").
		Build()

	resp, err := fc.client.Im.V1.MessageResource.Get(fc.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get image resource: %w", err)
	}

	if !resp.Success() {
		return nil, fmt.Errorf("failed to get image: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	if resp.File == nil {
		return nil, fmt.Errorf("image data is nil")
	}

	imageData, err := io.ReadAll(resp.File)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, nil
}

func (fc *FeishuChannel) getWorkspace() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp"
	}
	return filepath.Join(homeDir, ".nanobotgo", "media")
}

func isShareCardType(msgType string) bool {
	return msgType == "share_chat" || msgType == "share_user" ||
		msgType == "interactive" || msgType == "share_calendar_event" ||
		msgType == "system" || msgType == "merge_forward"
}

func extractShareCardContent(contentJSON string, msgType string) string {
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return fmt.Sprintf("[%s]", msgType)
	}

	switch msgType {
	case "share_chat":
		if chatID, ok := content["chat_id"].(string); ok {
			return fmt.Sprintf("[shared chat: %s]", chatID)
		}
	case "share_user":
		if userID, ok := content["user_id"].(string); ok {
			return fmt.Sprintf("[shared user: %s]", userID)
		}
	case "system":
		return "[system message]"
	case "merge_forward":
		return "[merged forward messages]"
	case "interactive":
		return extractInteractiveContent(content)
	case "share_calendar_event":
		if eventKey, ok := content["event_key"].(string); ok {
			return fmt.Sprintf("[shared calendar event: %s]", eventKey)
		}
	}

	return fmt.Sprintf("[%s]", msgType)
}

func extractInteractiveContent(content map[string]interface{}) string {
	var parts []string

	if title, ok := content["title"].(string); ok && title != "" {
		parts = append(parts, fmt.Sprintf("title: %s", title))
	} else if titleMap, ok := content["title"].(map[string]interface{}); ok {
		if titleContent, ok := titleMap["content"].(string); ok && titleContent != "" {
			parts = append(parts, fmt.Sprintf("title: %s", titleContent))
		} else if titleText, ok := titleMap["text"].(string); ok && titleText != "" {
			parts = append(parts, fmt.Sprintf("title: %s", titleText))
		}
	}

	if elements, ok := content["elements"].([]interface{}); ok {
		for _, el := range elements {
			if elMap, ok := el.(map[string]interface{}); ok {
				parts = append(parts, extractElementContent(elMap)...)
			}
		}
	}

	if card, ok := content["card"].(map[string]interface{}); ok {
		parts = append(parts, extractInteractiveContent(card))
	}

	if len(parts) == 0 {
		return "[interactive]"
	}

	return strings.Join(parts, "\n")
}

func extractElementContent(element map[string]interface{}) []string {
	var parts []string

	tag, _ := element["tag"].(string)

	switch tag {
	case "markdown", "lark_md":
		if content, ok := element["content"].(string); ok && content != "" {
			parts = append(parts, content)
		}
	case "div":
		if text, ok := element["text"].(string); ok && text != "" {
			parts = append(parts, text)
		} else if textMap, ok := element["text"].(map[string]interface{}); ok {
			if textContent, ok := textMap["content"].(string); ok && textContent != "" {
				parts = append(parts, textContent)
			} else if textContent, ok := textMap["text"].(string); ok && textContent != "" {
				parts = append(parts, textContent)
			}
		}
		if fields, ok := element["fields"].([]interface{}); ok {
			for _, f := range fields {
				if fieldMap, ok := f.(map[string]interface{}); ok {
					if fieldText, ok := fieldMap["text"].(map[string]interface{}); ok {
						if c, ok := fieldText["content"].(string); ok && c != "" {
							parts = append(parts, c)
						}
					}
				}
			}
		}
	case "a":
		if href, ok := element["href"].(string); ok && href != "" {
			parts = append(parts, fmt.Sprintf("link: %s", href))
		}
		if text, ok := element["text"].(string); ok && text != "" {
			parts = append(parts, text)
		}
	case "button":
		if text, ok := element["text"].(string); ok && text != "" {
			parts = append(parts, text)
		} else if textMap, ok := element["text"].(map[string]interface{}); ok {
			if c, ok := textMap["content"].(string); ok && c != "" {
				parts = append(parts, c)
			}
		}
		if url, ok := element["url"].(string); ok && url != "" {
			parts = append(parts, fmt.Sprintf("link: %s", url))
		}
	case "img":
		if alt, ok := element["alt"].(string); ok && alt != "" {
			parts = append(parts, alt)
		} else {
			parts = append(parts, "[image]")
		}
	case "note":
		if elements, ok := element["elements"].([]interface{}); ok {
			for _, ne := range elements {
				if neMap, ok := ne.(map[string]interface{}); ok {
					parts = append(parts, extractElementContent(neMap)...)
				}
			}
		}
	case "column_set":
		if columns, ok := element["columns"].([]interface{}); ok {
			for _, col := range columns {
				if colMap, ok := col.(map[string]interface{}); ok {
					if elements, ok := colMap["elements"].([]interface{}); ok {
						for _, ce := range elements {
							if ceMap, ok := ce.(map[string]interface{}); ok {
								parts = append(parts, extractElementContent(ceMap)...)
							}
						}
					}
				}
			}
		}
	case "plain_text":
		if content, ok := element["content"].(string); ok && content != "" {
			parts = append(parts, content)
		}
	default:
		if elements, ok := element["elements"].([]interface{}); ok {
			for _, ne := range elements {
				if neMap, ok := ne.(map[string]interface{}); ok {
					parts = append(parts, extractElementContent(neMap)...)
				}
			}
		}
	}

	return parts
}

func (fc *FeishuChannel) handlePostMessage(contentJSON string, messageID string) (string, []string) {
	var content map[string]interface{}
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		return "", nil
	}

	var root map[string]interface{}
	if post, ok := content["post"].(map[string]interface{}); ok {
		root = post
	} else {
		root = content
	}

	var text string
	var imageKeys []string

	locales := []string{"zh_cn", "en_us", "ja_jp"}
	for _, locale := range locales {
		if localeData, ok := root[locale].(map[string]interface{}); ok {
			text, imageKeys = fc.parsePostBlock(localeData)
			if text != "" || len(imageKeys) > 0 {
				break
			}
		}
	}

	if text == "" && len(imageKeys) == 0 {
		for _, v := range root {
			if vMap, ok := v.(map[string]interface{}); ok {
				text, imageKeys = fc.parsePostBlock(vMap)
				if text != "" || len(imageKeys) > 0 {
					break
				}
			}
		}
	}

	var media []string
	for _, imageKey := range imageKeys {
		if imagePath, err := fc.saveImage(messageID, imageKey); err == nil {
			media = append(media, imagePath)
		}
	}

	return text, media
}

func (fc *FeishuChannel) parsePostBlock(block map[string]interface{}) (string, []string) {
	var texts []string
	var images []string

	if title, ok := block["title"].(string); ok && title != "" {
		texts = append(texts, title)
	}

	if content, ok := block["content"].([]interface{}); ok {
		for _, row := range content {
			if rowList, ok := row.([]interface{}); ok {
				for _, el := range rowList {
					if elMap, ok := el.(map[string]interface{}); ok {
						tag, _ := elMap["tag"].(string)
						if tag == "text" || tag == "a" {
							if text, ok := elMap["text"].(string); ok && text != "" {
								texts = append(texts, text)
							}
						} else if tag == "at" {
							if userName, ok := elMap["user_name"].(string); ok && userName != "" {
								texts = append(texts, fmt.Sprintf("@%s", userName))
							} else {
								texts = append(texts, "@user")
							}
						} else if tag == "img" {
							if imageKey, ok := elMap["image_key"].(string); ok && imageKey != "" {
								images = append(images, imageKey)
							}
						}
					}
				}
			}
		}
	}

	result := strings.Join(texts, " ")
	return result, images
}
