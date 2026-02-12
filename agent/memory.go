package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nanobotgo/utils"
)

type MemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	utils.EnsureDir(memoryDir)

	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: filepath.Join(memoryDir, "MEMORY.md"),
	}
}

func (ms *MemoryStore) GetTodayFile() string {
	return filepath.Join(ms.memoryDir, utils.TodayDate()+".md")
}

func (ms *MemoryStore) ReadToday() string {
	todayFile := ms.GetTodayFile()
	if _, err := os.Stat(todayFile); os.IsNotExist(err) {
		return ""
	}

	content, err := os.ReadFile(todayFile)
	if err != nil {
		return ""
	}
	return string(content)
}

func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.GetTodayFile()

	var existing string
	if _, err := os.Stat(todayFile); err == nil {
		content, _ := os.ReadFile(todayFile)
		existing = string(content)
	}

	var newContent string
	if len(existing) > 0 {
		newContent = existing + "\n" + content
	} else {
		header := fmt.Sprintf("# %s\n\n", utils.TodayDate())
		newContent = header + content
	}

	return os.WriteFile(todayFile, []byte(newContent), 0644)
}

func (ms *MemoryStore) ReadLongTerm() string {
	if _, err := os.Stat(ms.memoryFile); os.IsNotExist(err) {
		return ""
	}

	content, err := os.ReadFile(ms.memoryFile)
	if err != nil {
		return ""
	}
	return string(content)
}

func (ms *MemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(ms.memoryFile, []byte(content), 0644)
}

func (ms *MemoryStore) GetRecentMemories(days int) string {
	var memories []string

	today := time.Now()

	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		filePath := filepath.Join(ms.memoryDir, dateStr+".md")

		if _, err := os.Stat(filePath); err == nil {
			content, _ := os.ReadFile(filePath)
			memories = append(memories, string(content))
		}
	}

	return strings.Join(memories, "\n\n---\n\n")
}

func (ms *MemoryStore) ListMemoryFiles() []string {
	var files []string

	entries, err := os.ReadDir(ms.memoryDir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	return files
}

func (ms *MemoryStore) GetMemoryContext() string {
	var parts []string

	longTerm := ms.ReadLongTerm()
	if longTerm != "" {
		parts = append(parts, "## Long-term Memory\n"+longTerm)
	}

	today := ms.ReadToday()
	if today != "" {
		parts = append(parts, "## Today's Notes\n"+today)
	}

	return strings.Join(parts, "\n\n")
}
