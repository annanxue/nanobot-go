package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/config"
	"github.com/sirupsen/logrus"
)

type WebChannel struct {
	*BaseChannelImpl
	config    *config.WebConfig
	wsManager *WebSocketManager
}

type WebSocketManager struct {
	bus         *bus.MessageBus
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
}

func NewWebChannel(cfg *config.WebConfig, bus *bus.MessageBus) (*WebChannel, error) {
	base := NewBaseChannel("web", cfg, bus)

	wsManager := &WebSocketManager{
		bus:         bus,
		connections: make(map[string]*websocket.Conn),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	return &WebChannel{
		BaseChannelImpl: base,
		config:          cfg,
		wsManager:       wsManager,
	}, nil
}

func (wc *WebChannel) Send(ctx context.Context, msg *bus.OutboundMessage) error {
	logrus.Infof("Sending message to web channel: %s", msg.Content)

	// Get WebSocket connection by ChatID (which is the connection ID)
	wc.wsManager.mu.RLock()
	ws, exists := wc.wsManager.connections[msg.ChatID]
	wc.wsManager.mu.RUnlock()

	if exists {
		response := map[string]interface{}{
			"success": true,
			"content": msg.Content,
		}
		if err := ws.WriteJSON(response); err != nil {
			fmt.Printf("Failed to send WebSocket message: %v\n", err)
			return err
		}
	}

	return nil
}

func (wc *WebChannel) HandleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) error {
	return wc.wsManager.HandleUpgrade(w, r)
}

func (wm *WebSocketManager) HandleUpgrade(w http.ResponseWriter, r *http.Request) error {
	// Upgrade HTTP connection to WebSocket
	ws, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade to WebSocket: %v\n", err)
		return err
	}
	defer ws.Close()

	// Generate unique connection ID using UUID
	connID := fmt.Sprintf("conn_%s", uuid.New().String())

	// Add connection to the list
	wm.mu.Lock()
	wm.connections[connID] = ws
	wm.mu.Unlock()

	// Process incoming messages from client
	for {
		// Read message from client
		_, message, err := ws.ReadMessage()
		if err != nil {
			// Remove connection from list
			wm.mu.Lock()
			delete(wm.connections, connID)
			wm.mu.Unlock()
			break
		}

		// Parse message
		var req struct {
			Type     string `json:"type"`
			Content  string `json:"content"`
			Image    string `json:"image,omitempty"`
			Filename string `json:"filename,omitempty"`
		}
		if err := json.Unmarshal(message, &req); err != nil {
			fmt.Printf("Failed to parse WebSocket message: %v\n", err)
			continue
		}

		// Handle chat message
		if req.Type == "chat" && req.Content != "" {
			var media []string
			if req.Image != "" {
				media = append(media, req.Image)
			}

			inboundMsg := &bus.InboundMessage{
				Channel:   "web",
				SenderID:  "web-user",
				ChatID:    connID,
				Content:   req.Content,
				Media:     media,
				Timestamp: time.Now(),
			}

			wm.bus.PublishInbound(inboundMsg)
		}
	}

	return nil
}
