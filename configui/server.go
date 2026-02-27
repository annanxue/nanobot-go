package configui

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nanobotgo/agent"
	"github.com/nanobotgo/bus"
	"github.com/nanobotgo/channels"
	"github.com/nanobotgo/config"
	"github.com/nanobotgo/cron"
	"github.com/nanobotgo/session"
)

var (
	addr = ":18080"
)

type Server struct {
	httpServer     *http.Server
	loader         *config.Loader
	configPath     string
	config         *config.Config
	cronService    *cron.CronService
	sessionManager *session.SessionManager
	skillsLoader   *agent.SkillsLoader
	messageBus     *bus.MessageBus
	mu             sync.RWMutex
	engine         *gin.Engine
	webChannel     *channels.WebChannel
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func NewServer(cfg *config.Config, configPath string, loader *config.Loader, cronService *cron.CronService, sessionManager *session.SessionManager, skillsLoader *agent.SkillsLoader, messageBus *bus.MessageBus, webChannel *channels.WebChannel, listenAddr string) *Server {
	if listenAddr != "" {
		addr = listenAddr
	}

	return &Server{
		config:         cfg,
		configPath:     configPath,
		loader:         loader,
		cronService:    cronService,
		sessionManager: sessionManager,
		skillsLoader:   skillsLoader,
		messageBus:     messageBus,
		webChannel:     webChannel,
	}
}

func (s *Server) loadConfig() (*config.Config, error) {
	if s.config != nil {
		return s.config, nil
	}
	return s.loader.Load()
}

func (s *Server) saveConfig(cfg *config.Config) error {
	return s.loader.Save(cfg)
}

func (s *Server) getConfigPath() string {
	if s.configPath != "" {
		return s.configPath
	}
	return s.loader.GetConfigPath()
}

func (s *Server) setupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	staticPath := "./configui/static"
	if _, err := os.Stat(staticPath); os.IsNotExist(err) {
		exePath, _ := os.Executable()
		dir := filepath.Dir(exePath)
		staticPath = filepath.Join(dir, "configui", "static")
	}
	engine.Static("/static", staticPath)

	templatePath := "./configui/templates"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		exePath, _ := os.Executable()
		dir := filepath.Dir(exePath)
		templatePath = filepath.Join(dir, "configui", "templates")
	}

	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"contains": strings.Contains,
	}).ParseGlob(templatePath + "/*"))
	engine.SetHTMLTemplate(tmpl)

	engine.GET("/", func(c *gin.Context) {
		cfg, err := s.loadConfig()
		if err != nil {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"Title":   s.getConfigPath(),
				"Config":  "{}",
				"Message": "加载配置失败: " + err.Error(),
			})
			return
		}

		cfgJSON, _ := json.Marshal(cfg)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Title":  s.getConfigPath(),
			"Config": string(cfgJSON),
		})
	})

	engine.GET("/chat", func(c *gin.Context) {
		c.HTML(http.StatusOK, "chat.html", nil)
	})

	engine.GET("/ws/chat", func(c *gin.Context) {
		if s.webChannel != nil {
			s.webChannel.HandleWebSocketUpgrade(c.Writer, c.Request)
		}
	})

	api := engine.Group("/api")
	{
		api.GET("/config", s.handleGetConfig)
		api.POST("/config", s.handleSaveConfig)
		api.POST("/restart", s.handleRestart)
		api.GET("/cron", s.handleGetCronJobs)
		api.DELETE("/cron/:id", s.handleDeleteCronJob)
		api.GET("/sessions", s.handleGetSessions)
		api.DELETE("/sessions/:key", s.handleDeleteSession)
		api.GET("/skills", s.handleGetSkills)
		api.POST("/chat", s.handleChat)
	}

	return engine
}

func (s *Server) handleGetConfig(c *gin.Context) {
	cfg, err := s.loadConfig()
	if err != nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: cfg})
}

func (s *Server) handleSaveConfig(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: err.Error()})
		return
	}

	var cfg config.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: err.Error()})
		return
	}

	if err := s.saveConfig(&cfg); err != nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: err.Error()})
		return
	}

	s.mu.Lock()
	s.config = &cfg
	s.mu.Unlock()

	c.JSON(http.StatusOK, APIResponse{Success: true})
}

func (s *Server) handleRestart(c *gin.Context) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Message: "服务重启成功"})
}

func (s *Server) handleGetCronJobs(c *gin.Context) {
	if s.cronService == nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Cron service not available"})
		return
	}
	jobs := s.cronService.ListJobs(true)
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: jobs})
}

func (s *Server) handleDeleteCronJob(c *gin.Context) {
	if s.cronService == nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Cron service not available"})
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Cron job ID is required"})
		return
	}
	deleted := s.cronService.RemoveJob(id)
	if !deleted {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Failed to delete cron job"})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Cron job deleted successfully"})
}

func (s *Server) handleGetSessions(c *gin.Context) {
	if s.sessionManager == nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Session manager not available"})
		return
	}
	sessions := s.sessionManager.ListSessions()
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: sessions})
}

func (s *Server) handleDeleteSession(c *gin.Context) {
	if s.sessionManager == nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Session manager not available"})
		return
	}
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Session key is required"})
		return
	}
	deleted := s.sessionManager.Delete(key)
	if !deleted {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Failed to delete session"})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Session deleted successfully"})
}

func (s *Server) handleGetSkills(c *gin.Context) {
	if s.skillsLoader == nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Skills loader not available"})
		return
	}
	skills := s.skillsLoader.ListSkills(false)
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: skills})
}

func (s *Server) handleChat(c *gin.Context) {
	if s.messageBus == nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Message bus not available"})
		return
	}

	var req struct {
		Content  string `json:"content"`
		Image    string `json:"image,omitempty"`
		Filename string `json:"filename,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, APIResponse{Success: false, Error: "Invalid request"})
		return
	}

	var media []string
	if req.Image != "" {
		media = append(media, req.Image)
	}

	inboundMsg := &bus.InboundMessage{
		Channel:   "web",
		SenderID:  "web-user",
		ChatID:    "web-chat",
		Content:   req.Content,
		Media:     media,
		Timestamp: now(),
	}

	s.messageBus.PublishInbound(inboundMsg)

	c.JSON(http.StatusOK, APIResponse{Success: true, Message: "Message sent"})
}

func now() time.Time {
	return time.Now()
}

func (s *Server) Start() error {
	engine := s.setupRoutes()
	s.engine = engine

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	fmt.Printf("🎨 配置管理界面启动成功: http://localhost%s\n", addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}
