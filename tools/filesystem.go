package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ReadFileTool struct {
	allowedDir string
}

func NewReadFileTool(allowedDir string) *ReadFileTool {
	return &ReadFileTool{allowedDir: allowedDir}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file at the given path."
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file path to read",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	filePath, err := resolvePath(path, t.allowedDir)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", path)
	}

	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		return "", fmt.Errorf("not a file: %s", path)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return string(content), nil
}

type WriteFileTool struct {
	allowedDir string
}

func NewWriteFileTool(allowedDir string) *WriteFileTool {
	return &WriteFileTool{allowedDir: allowedDir}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file at the given path. Creates parent directories if needed."
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file path to write to",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The content to write",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}

	filePath, err := resolvePath(path, t.allowedDir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("error creating directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

type EditFileTool struct {
	allowedDir string
}

func NewEditFileTool(allowedDir string) *EditFileTool {
	return &EditFileTool{allowedDir: allowedDir}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file."
}

func (t *EditFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The file path to edit",
			},
			"old_text": map[string]interface{}{
				"type":        "string",
				"description": "The exact text to find and replace",
			},
			"new_text": map[string]interface{}{
				"type":        "string",
				"description": "The text to replace with",
			},
		},
		"required": []string{"path", "old_text", "new_text"},
	}
}

func (t *EditFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	oldText, ok := params["old_text"].(string)
	if !ok {
		return "", fmt.Errorf("old_text is required")
	}

	newText, ok := params["new_text"].(string)
	if !ok {
		return "", fmt.Errorf("new_text is required")
	}

	filePath, err := resolvePath(path, t.allowedDir)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", path)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, oldText) {
		return "", fmt.Errorf("old_text not found in file. Make sure it matches exactly.")
	}

	count := strings.Count(contentStr, oldText)
	if count > 1 {
		return "", fmt.Errorf("old_text appears %d times. Please provide more context to make it unique.", count)
	}

	newContent := strings.Replace(contentStr, oldText, newText, 1)
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return fmt.Sprintf("Successfully edited %s", path), nil
}

type ListDirTool struct {
	allowedDir string
}

func NewListDirTool(allowedDir string) *ListDirTool {
	return &ListDirTool{allowedDir: allowedDir}
}

func (t *ListDirTool) Name() string {
	return "list_dir"
}

func (t *ListDirTool) Description() string {
	return "List the contents of a directory."
}

func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "The directory path to list",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	dirPath, err := resolvePath(path, t.allowedDir)
	if err != nil {
		return "", err
	}

	if info, err := os.Stat(dirPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", path)
		}
		return "", fmt.Errorf("error accessing path: %w", err)
	} else if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("error listing directory: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Sprintf("Directory %s is empty", path), nil
	}

	var items []string
	for _, entry := range entries {
		prefix := "📄 "
		if entry.IsDir() {
			prefix = "📁 "
		}
		items = append(items, prefix+entry.Name())
	}

	sort.Strings(items)
	return strings.Join(items, "\n"), nil
}

func resolvePath(path string, allowedDir string) (string, error) {
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("error resolving path: %w", err)
	}

	if allowedDir != "" {
		allowedAbs, err := filepath.Abs(allowedDir)
		if err != nil {
			return "", fmt.Errorf("error resolving allowed directory: %w", err)
		}
		if !strings.HasPrefix(absPath, allowedAbs) {
			return "", fmt.Errorf("path %s is outside allowed directory %s", path, allowedDir)
		}
	}

	return absPath, nil
}
