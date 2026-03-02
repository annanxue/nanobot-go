package utils

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func EnsureDir(path string) string {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		panic(err)
	}
	return path
}

func GetDataPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return EnsureDir(filepath.Join(homeDir, ".nanobotgo"))
}

func GetWorkspacePath(workspace string) string {
	var path string
	if workspace != "" {
		path = filepath.Clean(workspace)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		path = filepath.Join(homeDir, ".nanobotgo", "workspace")
	}
	return EnsureDir(path)
}

func GetSessionsPath() string {
	return EnsureDir(filepath.Join(GetDataPath(), "sessions"))
}

func GetMemoryPath(workspace string) string {
	ws := workspace
	if ws == "" {
		ws = GetWorkspacePath("")
	}
	return EnsureDir(filepath.Join(ws, "memory"))
}

func GetSkillsPath(workspace string) string {
	ws := workspace
	if ws == "" {
		ws = GetWorkspacePath("")
	}
	return EnsureDir(filepath.Join(ws, "skills"))
}

func TodayDate() string {
	return time.Now().Format("2006-01-02")
}

func Timestamp() string {
	return time.Now().Format(time.RFC3339)
}

func TruncateString(s string, maxLen int, suffix string) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-len(suffix)] + suffix
}

func SafeFilename(name string) string {
	unsafe := "<>:\"/\\|?*"
	for _, char := range unsafe {
		name = strings.ReplaceAll(name, string(char), "_")
	}
	return strings.TrimSpace(name)
}

func ParseSessionKey(key string) (string, string, error) {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return "", "", &ParseError{Key: key, Msg: "invalid session key"}
	}
	return parts[0], parts[1], nil
}

type ParseError struct {
	Key string
	Msg string
}

func (e *ParseError) Error() string {
	return e.Msg + ": " + e.Key
}
