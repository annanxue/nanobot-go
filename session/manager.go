package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nanobotgo/utils"
	"github.com/sirupsen/logrus"
)

type SessionMessage struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp string                 `json:"timestamp"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

type Session struct {
	Key       string                 `json:"key"`
	Messages  []SessionMessage       `json:"messages"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func (s *Session) AddMessage(role, content string, extra map[string]interface{}) {
	msg := SessionMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now().Format(time.RFC3339),
		Extra:     extra,
	}
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
}

func (s *Session) GetHistory(maxMessages int) []map[string]interface{} {
	start := 0
	if len(s.Messages) > maxMessages {
		start = len(s.Messages) - maxMessages
	}

	history := make([]map[string]interface{}, 0, len(s.Messages)-start)
	for i := start; i < len(s.Messages); i++ {
		msg := s.Messages[i]
		history = append(history, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}
	return history
}

func (s *Session) Clear() {
	s.Messages = []SessionMessage{}
	s.UpdatedAt = time.Now()
}

type SessionManager struct {
	workspace   string
	sessionsDir string
	cache       map[string]*Session
	cacheMutex  sync.RWMutex
}

func NewSessionManager(workspace string) *SessionManager {
	sessionsDir := filepath.Join(utils.GetDataPath(), "sessions")
	utils.EnsureDir(sessionsDir)

	return &SessionManager{
		workspace:   workspace,
		sessionsDir: sessionsDir,
		cache:       make(map[string]*Session),
	}
}

func (sm *SessionManager) getSessionPath(key string) string {
	safeKey := utils.SafeFilename(key)
	safeKey = strings.ReplaceAll(safeKey, ":", "_")
	return filepath.Join(sm.sessionsDir, safeKey+".jsonl")
}

func (sm *SessionManager) GetOrCreate(key string) *Session {
	sm.cacheMutex.RLock()
	session, exists := sm.cache[key]
	sm.cacheMutex.RUnlock()

	if exists {
		return session
	}

	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	session = sm.load(key)
	if session == nil {
		session = &Session{
			Key:       key,
			Messages:  []SessionMessage{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	}

	sm.cache[key] = session
	return session
}

func (sm *SessionManager) load(key string) *Session {
	path := sm.getSessionPath(key)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		logrus.WithError(err).Warnf("Failed to load session %s", key)
		return nil
	}
	defer file.Close()

	var messages []SessionMessage
	var metadata map[string]interface{}
	var createdAt, updatedAt time.Time

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var data map[string]interface{}
		if err := decoder.Decode(&data); err != nil {
			continue
		}

		if typ, ok := data["_type"].(string); ok && typ == "metadata" {
			if ct, ok := data["created_at"].(string); ok {
				createdAt, _ = time.Parse(time.RFC3339, ct)
			}
			if ut, ok := data["updated_at"].(string); ok {
				updatedAt, _ = time.Parse(time.RFC3339, ut)
			}
			if md, ok := data["metadata"].(map[string]interface{}); ok {
				metadata = md
			}
		} else {
			msg := SessionMessage{
				Role:      data["role"].(string),
				Content:   data["content"].(string),
				Timestamp: data["timestamp"].(string),
			}
			if extra, ok := data["extra"].(map[string]interface{}); ok {
				msg.Extra = extra
			}
			messages = append(messages, msg)
		}
	}

	return &Session{
		Key:       key,
		Messages:  messages,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Metadata:  metadata,
	}
}

func (sm *SessionManager) Save(session *Session) {
	path := sm.getSessionPath(session.Key)
	file, err := os.Create(path)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to save session %s", session.Key)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	metadata := map[string]interface{}{
		"_type":      "metadata",
		"created_at": session.CreatedAt.Format(time.RFC3339),
		"updated_at": session.UpdatedAt.Format(time.RFC3339),
		"metadata":   session.Metadata,
	}
	encoder.Encode(metadata)

	for _, msg := range session.Messages {
		data := map[string]interface{}{
			"role":      msg.Role,
			"content":   msg.Content,
			"timestamp": msg.Timestamp,
		}
		if msg.Extra != nil {
			data["extra"] = msg.Extra
		}
		encoder.Encode(data)
	}

	sm.cacheMutex.Lock()
	sm.cache[session.Key] = session
	sm.cacheMutex.Unlock()
}

func (sm *SessionManager) Delete(key string) bool {
	sm.cacheMutex.Lock()
	defer sm.cacheMutex.Unlock()

	delete(sm.cache, key)

	path := sm.getSessionPath(key)
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
		return true
	}
	return false
}

type SessionInfo struct {
	Key       string `json:"key"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Path      string `json:"path"`
}

func (sm *SessionManager) ListSessions() []SessionInfo {
	var sessions []SessionInfo

	files, err := os.ReadDir(sm.sessionsDir)
	if err != nil {
		return sessions
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") {
			continue
		}

		path := filepath.Join(sm.sessionsDir, file.Name())
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		var createdAt, updatedAt time.Time
		decoder := json.NewDecoder(f)
		for decoder.More() {
			var data map[string]interface{}
			if err := decoder.Decode(&data); err != nil {
				continue
			}

			if typ, ok := data["_type"].(string); ok && typ == "metadata" {
				if ct, ok := data["created_at"].(string); ok {
					createdAt, _ = time.Parse(time.RFC3339, ct)
				}
				if ut, ok := data["updated_at"].(string); ok {
					updatedAt, _ = time.Parse(time.RFC3339, ut)
				}
				break
			}
		}
		f.Close()

		sessions = append(sessions, SessionInfo{
			Key:       strings.ReplaceAll(file.Name()[:len(file.Name())-6], "_", ":"),
			CreatedAt: createdAt.Format(time.RFC3339),
			UpdatedAt: updatedAt.Format(time.RFC3339),
			Path:      path,
		})
	}

	return sessions
}
