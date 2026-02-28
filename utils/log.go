package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	// Initialize logger
	Log = logrus.New()

	// Enable caller reporting
	Log.SetReportCaller(true)

	// Set level
	Log.SetLevel(logrus.DebugLevel)

	// Set custom formatter
	Log.SetFormatter(&CustomFormatter{})

	// Create log directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		Log.Errorf("Failed to create log directory: %v", err)
	}

	// Create log file with timestamp
	logFile, err := os.Create("logs/nanobot_" + time.Now().Format("20060102_150405") + ".log")
	if err != nil {
		Log.Errorf("Failed to create log file: %v", err)
		// Fallback to stdout only if file creation fails
		Log.SetOutput(os.Stdout)
	} else {
		// Write to both stdout and file
		Log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}
}

// 获取当前协程 ID（安全实现，兼容各 Go 版本）
func getGoroutineID() int {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	// 解析栈信息中的协程 ID，格式："goroutine 123 ["
	buf = bytes.TrimPrefix(buf, []byte("goroutine "))
	idStr := string(buf[:bytes.IndexByte(buf, ' ')])
	id, _ := strconv.Atoi(idStr)
	return id
}

// CustomFormatter implements logrus.Formatter interface
// Format logs as: [2026-02-28 16:00:01.123] [DEBUG ] [PID:12345 | GID:1] [main.go:95] - 调试信息：用户请求参数 -> {"user_id": 123, "action": "login"}
type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Get PID and GID
	// pid := os.Getpid()
	// gid := syscall.Getgid()
	goroutineid := getGoroutineID()

	// Format timestamp with milliseconds
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000")

	// Format level (5 characters, padded with spaces)
	level := strings.ToUpper(entry.Level.String())
	// Map WARNING to WARN
	if level == "WARNING" {
		level = "WARN"
	}
	level = fmt.Sprintf("%-5s", level)

	// Get file and line number
	file := ""
	line := 0
	if entry.HasCaller() {
		file = filepath.Base(entry.Caller.File)
		line = entry.Caller.Line
	}

	// Format message with fields
	message := entry.Message
	if len(entry.Data) > 0 {
		message += " -> " + fmt.Sprintf("%v", entry.Data)
	}

	// Build log line
	logLine := fmt.Sprintf("[%s] [%s] [GID:%d] [%s:%d] - %s\n",
		timestamp,
		level,
		goroutineid,
		file,
		line,
		message)

	return []byte(logLine), nil
}

// WithFields returns a logger with the given fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}
