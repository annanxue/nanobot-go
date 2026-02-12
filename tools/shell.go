package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ExecTool struct {
	timeout             int
	workingDir          string
	denyPatterns        []string
	allowPatterns       []string
	restrictToWorkspace bool
}

func NewExecTool(timeout int, workingDir string, denyPatterns, allowPatterns []string, restrictToWorkspace bool) *ExecTool {
	defaultDenyPatterns := []string{
		`\brm\s+-[rf]{1,2}\b`,
		`\bdel\s+/[fq]\b`,
		`\brmdir\s+/s\b`,
		`\b(format|mkfs|diskpart)\b`,
		`\bdd\s+if=`,
		`>\s*/dev/sd`,
		`\b(shutdown|reboot|poweroff)\b`,
		`:\(\)\s*\{.*\};\s*:`,
	}

	if denyPatterns == nil {
		denyPatterns = defaultDenyPatterns
	}

	return &ExecTool{
		timeout:             timeout,
		workingDir:          workingDir,
		denyPatterns:        denyPatterns,
		allowPatterns:       allowPatterns,
		restrictToWorkspace: restrictToWorkspace,
	}
}

func (t *ExecTool) Name() string {
	return "exec"
}

func (t *ExecTool) Description() string {
	return "Execute a shell command and return its output. Use with caution."
}

func (t *ExecTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Optional working directory for the command",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ExecTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command is required")
	}

	workingDir := t.workingDir
	if wd, ok := params["working_dir"].(string); ok {
		workingDir = wd
	}

	if workingDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error getting working directory: %w", err)
		}
		workingDir = wd
	}

	if guardError := t.guardCommand(command, workingDir); guardError != "" {
		return "", fmt.Errorf(guardError)
	}

	var cmd *exec.Cmd
	if strings.Contains(strings.ToLower(command), "&&") || strings.Contains(strings.ToLower(command), "||") {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	} else {
		parts := strings.Fields(command)
		if len(parts) > 0 {
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		} else {
			return "", fmt.Errorf("empty command")
		}
	}

	if workingDir != "" {
		cmd.Dir = workingDir
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(t.timeout)*time.Second)
	defer cancel()

	cmd = exec.CommandContext(timeoutCtx, cmd.Path, cmd.Args[1:]...)

	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := exitErr.Stderr
			var outputParts []string
			if len(stdout) > 0 {
				outputParts = append(outputParts, string(stdout))
			}
			if len(stderr) > 0 {
				outputParts = append(outputParts, "STDERR:\n"+string(stderr))
			}
			outputParts = append(outputParts, fmt.Sprintf("\nExit code: %d", exitErr.ExitCode()))
			return strings.Join(outputParts, "\n"), nil
		}
		return "", fmt.Errorf("error executing command: %w", err)
	}

	result := string(stdout)
	if len(result) > 10000 {
		result = result[:10000] + fmt.Sprintf("\n... (truncated, %d more chars)", len(result)-10000)
	}

	if result == "" {
		return "(no output)", nil
	}

	return result, nil
}

func (t *ExecTool) guardCommand(command, cwd string) string {
	cmd := strings.TrimSpace(command)
	lower := strings.ToLower(cmd)

	for _, pattern := range t.denyPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(lower) {
			return "Error: Command blocked by safety guard (dangerous pattern detected)"
		}
	}

	if len(t.allowPatterns) > 0 {
		allowed := false
		for _, pattern := range t.allowPatterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			if re.MatchString(lower) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "Error: Command blocked by safety guard (not in allowlist)"
		}
	}

	if t.restrictToWorkspace {
		if strings.Contains(cmd, "..\\") || strings.Contains(cmd, "../") {
			return "Error: Command blocked by safety guard (path traversal detected)"
		}

		cwdPath, err := filepath.Abs(cwd)
		if err != nil {
			return ""
		}

		winPathRe := regexp.MustCompile(`[A-Za-z]:\\[^\\\"']+`)
		posixPathRe := regexp.MustCompile(`(?:^|[\s|>])(/[^\s\"'>]+)`)

		allMatches := append(winPathRe.FindAllString(cmd, -1), posixPathRe.FindAllString(cmd, -1)...)

		for _, raw := range allMatches {
			p := filepath.Clean(strings.TrimSpace(raw))
			absPath, err := filepath.Abs(p)
			if err != nil {
				continue
			}

			if filepath.IsAbs(absPath) {
				parent := filepath.Dir(absPath)
				if parent != cwdPath && absPath != cwdPath {
					return "Error: Command blocked by safety guard (path outside working dir)"
				}
			}
		}
	}

	return ""
}
